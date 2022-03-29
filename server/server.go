package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/wangyuqi0706/yqkv/client"
	"github.com/wangyuqi0706/yqkv/session"
	"github.com/wangyuqi0706/yqkv/session/sessionpb"
	"github.com/wangyuqi0706/yqkv/shardctl"
	"github.com/wangyuqi0706/yqkv/shardctl/shardctlpb"
	"github.com/wangyuqi0706/yqkv/yqkvpb"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.etcd.io/etcd/pkg/v3/wait"

	"github.com/wangyuqi0706/yqkv/raft"
)

type StateMachine struct {
	Shards        map[int64]*Shard
	SessionMap    *session.SessionMap
	AppliedIndex  uint64
	CurrentConfig *shardctl.Config
	LastConfig    *shardctl.Config
}

func (s *StateMachine) DeepCopy() *StateMachine {
	copied := &StateMachine{
		Shards:       map[int64]*Shard{},
		SessionMap:   s.SessionMap.DeepCopy(),
		AppliedIndex: s.AppliedIndex,
	}
	*copied.LastConfig = *s.LastConfig
	*copied.CurrentConfig = *s.CurrentConfig
	for i, shard := range s.Shards {
		copied.Shards[i] = shard
	}
	return copied
}

type ShardKV struct {
	mu           *sync.RWMutex
	me           uint64
	rf           *raft.Raft
	applyCh      chan raft.ApplyMsg
	make_end     func(string) yqkvpb.ShardKVClient
	gid          int64
	ctrlers      []shardctlpb.ShardctlClient
	maxraftstate int64 // snapshot if log grows this big

	// Your definitions here.
	waitApply wait.WaitTime

	// should be closed when not migrating
	migrating uint32

	*StateMachine

	controller *shardctl.Clerk
	clerk      *client.Clerk

	stop       chan struct{}
	snapshotCh chan struct{}

	logger *log.Logger
}

func (kv *ShardKV) Monitor(action func(), timeout time.Duration) {
	for {
		select {
		case <-kv.stop:
			return
		default:
			if _, isLeader := kv.rf.GetState(); isLeader {
				action()
			}
			time.Sleep(timeout)
		}
	}
}

func (kv *ShardKV) Execute(args *yqkvpb.Args, reply *yqkvpb.Reply) {
	kv.mu.Lock()
	if kv.SessionMap.IsDuplicate(args.SessionHeader) {
		if resp, err := kv.SessionMap.GetResponse(args.SessionHeader); err != nil {
			reply.Status = yqkvpb.ERR_BAD_SESSION
			reply.Message = err.Error()
		} else {
			*reply = *resp.(*yqkvpb.Reply)
		}
		kv.mu.Unlock()
		return
	}
	kv.mu.Unlock()
	data, _ := args.Marshal()
	index, _, leader := kv.rf.Start(data)
	if !leader {
		reply.Status = yqkvpb.ERR_WRONG_LEADER
		return
	}
	//kv.logger.Printf("RECEIVE args=%v", args)
	<-kv.waitApply.Wait(index)
	kv.mu.Lock()
	defer kv.mu.Unlock()
	response, err := kv.SessionMap.GetResponse(args.SessionHeader)
	if err != nil {
		reply.Status = yqkvpb.ERR_BAD_SESSION
		reply.Message = err.Error()
		return
	}
	*reply = *response.(*yqkvpb.Reply)
	//kv.logger.Printf("RETURN %v index=%v request=%v reply=%v", args.Type, index, args.SessionHeader, reply)
}

func (kv *ShardKV) PullShards(args *yqkvpb.Args, reply *yqkvpb.Reply) {
	if _, lead := kv.rf.GetState(); !lead {
		reply.Status = yqkvpb.ERR_WRONG_LEADER
		return
	}
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if kv.CurrentConfig.Num < args.ConfigNum {
		reply.Status = yqkvpb.ERR_NOT_READY
		return
	}
	var resp *yqkvpb.PullShardsResponse
	resp = &yqkvpb.PullShardsResponse{}
	for _, shardID := range args.ShardIDs {
		resp.Shards = append(resp.Shards, kv.Shards[shardID].ProtoMessage())
	}
	resp.ServerSessions = kv.SessionMap.ProtoMessage()
	resp.ConfigNum = args.ConfigNum
	reply.Data, _ = resp.Marshal()
	reply.Status = yqkvpb.OK
	//kv.logger.Printf("RETURN PullShards args=%v reply=%v", args, reply.Value)
}

func (kv *ShardKV) DeleteShards(args *yqkvpb.Args, reply *yqkvpb.Reply) {
	if _, lead := kv.rf.GetState(); !lead {
		reply.Status = yqkvpb.ERR_WRONG_LEADER
		return
	}
	kv.mu.RLock()
	if kv.CurrentConfig.Num > args.ConfigNum {
		reply.Status = yqkvpb.OK
		kv.mu.RUnlock()
		return
	}
	kv.mu.RUnlock()
	kv.Execute(args, reply)
	reply.Status = yqkvpb.OK
}

func (kv *ShardKV) configureAction() {
	// do not fetch when migrating
	migrating := false
	kv.mu.RLock()
	for _, shard := range kv.Shards {
		if shard.Status() != yqkvpb.SERVING {
			migrating = true
			break
		}
	}
	currentNum := kv.CurrentConfig.Num
	kv.mu.RUnlock()
	if migrating {
		return
	}
	nextConfig := kv.controller.Query(currentNum + 1)
	if nextConfig.Num == currentNum+1 {
		args := &yqkvpb.Args{
			SessionHeader: kv.clerk.NextHeader(),
			Type:          yqkvpb.RECONFIGURE,
			Key:           "",
			NextConfig:    nextConfig,
		}
		kv.logger.Printf("EXECUTE CONFIG config=%v", nextConfig)
		kv.Execute(args, &yqkvpb.Reply{})
		kv.logger.Printf("NEW CONFIG config=%v", nextConfig)
	}
}

func (kv *ShardKV) migratingAction() {
	kv.mu.RLock()
	wg := &sync.WaitGroup{}
	gid2ShardIDs := kv.getGid2ShardsOnLastConfigByStatus(yqkvpb.PULLING)
	for gid, shardIDs := range gid2ShardIDs {
		wg.Add(1)
		go func(gid int64, shardIDs []int64, configNum int64) {
			kv.logger.Printf("PULL SHARDS %v from gid=%v configNum=%v", shardIDs, gid, configNum)
			resp := kv.clerk.PullShards(gid, configNum, shardIDs)
			kv.Execute(&yqkvpb.Args{Type: yqkvpb.INSERT_SHARDS, PulledShards: resp, SessionHeader: kv.clerk.NextHeader()}, &yqkvpb.Reply{})
			wg.Done()
			kv.logger.Printf("PULL SUCCESS SHARDS %v from gid=%v configNum=%v", shardIDs, gid, configNum)
		}(gid, shardIDs, kv.CurrentConfig.Num)
	}
	kv.mu.RUnlock()
	wg.Wait()
	return
}

func (kv *ShardKV) checkEntryInCurrentTermAction() {
	if !kv.rf.HasCurrentTermLog() {
		kv.Execute(&yqkvpb.Args{Type: yqkvpb.EMPTY, SessionHeader: kv.clerk.NextHeader()}, &yqkvpb.Reply{})
	}
}

// Return last configuration's gid -> []shardID by current configuration's ShardStatus
// Calling this method when you need to pull/delete shards from other replicas
func (kv *ShardKV) getGid2ShardsOnLastConfigByStatus(status yqkvpb.ShardStatus) map[int64][]int64 {
	gid2ShardIDs := make(map[int64][]int64)
	for id, shard := range kv.Shards {
		if shard.Status() == status {
			gid := kv.LastConfig.Shards[id]
			if _, ok := gid2ShardIDs[gid]; ok {
				gid2ShardIDs[gid] = append(gid2ShardIDs[gid], shard.ID())
			} else {
				gid2ShardIDs[gid] = []int64{id}
			}
		}
	}
	return gid2ShardIDs
}

func (kv *ShardKV) gcAction() {
	kv.mu.RLock()
	wg := &sync.WaitGroup{}
	gid2ShardIDs := kv.getGid2ShardsOnLastConfigByStatus(yqkvpb.GC)
	for gid, shardIDs := range gid2ShardIDs {
		wg.Add(1)
		go func(gid int64, shardIDs []int64, configNum int64) {
			kv.logger.Printf("DELETE REMOTE SHARDS %v at gid=%v configNum=%v", shardIDs, gid, configNum)
			kv.clerk.DeleteShards(gid, configNum, shardIDs)
			kv.Execute(&yqkvpb.Args{Type: yqkvpb.DELETE_SHARDS, ConfigNum: configNum, ShardIDs: shardIDs, SessionHeader: kv.clerk.NextHeader()}, &yqkvpb.Reply{})
			wg.Done()
			kv.logger.Printf("DELETE REMOTE SHARDS SUCCESS")
		}(gid, shardIDs, kv.CurrentConfig.Num)
	}
	kv.mu.RUnlock()
	wg.Wait()
	return
}

func (kv *ShardKV) run() {
	for {
		select {
		case msg := <-kv.applyCh:
			if msg.CommandValid && msg.Command != nil {
				kv.apply(msg)
				if kv.shouldSnapshot() {
					kv.createSnapshot()
				}
			} else if msg.SnapshotValid {
				if kv.rf.CondInstallSnapshot(msg.SnapshotTerm, msg.SnapshotIndex, msg.Snapshot) {
					kv.installSnapshot(msg.Snapshot)
				}
			}
		case <-kv.stop:
			return
		}
	}
}

func (kv *ShardKV) apply(msg raft.ApplyMsg) {
	args := yqkvpb.Args{}
	_ = proto.Unmarshal(msg.Command, &args)
	//kv.logger.Printf("RUN index=%v args=%v", msg.CommandIndex, args)
	if kv.SessionMap.IsDuplicate(args.SessionHeader) {
		//kv.logger.Printf("DUPLICATE OP args=%v", args)
		atomic.StoreUint64(&kv.AppliedIndex, msg.CommandIndex)
		kv.waitApply.Trigger(atomic.LoadUint64(&kv.AppliedIndex))
		return
	}
	var reply = &yqkvpb.Reply{Status: yqkvpb.OK}
	switch args.Type {
	case yqkvpb.EMPTY:
		reply = kv.applyEmpty()
	case yqkvpb.GET:
		reply = kv.applyGet(args.Key)
	case yqkvpb.PUT:
		reply = kv.applyPut(args.Key, args.Value)
	case yqkvpb.APPEND:
		reply = kv.applyAppend(args.Key, args.Value)
	case yqkvpb.PULL_SHARDS:
		reply = kv.applyInsertShards(args.PulledShards)
	case yqkvpb.DELETE_SHARDS:
		reply = kv.applyDeleteShards(args.ConfigNum, args.ShardIDs)
	case yqkvpb.RECONFIGURE:
		reply = kv.applyReconfigure(args.NextConfig)
	}
	if !kv.SessionMap.HasSession(args.SessionHeader.ID) {
		_, _ = kv.SessionMap.CreateSession(args.SessionHeader.ID)
	}
	_ = kv.SessionMap.SetResponse(args.SessionHeader, reply)
	atomic.StoreUint64(&kv.AppliedIndex, msg.CommandIndex)
	kv.waitApply.Trigger(atomic.LoadUint64(&kv.AppliedIndex))

}

func (kv *ShardKV) applyGet(key string) *yqkvpb.Reply {
	shard := client.Key2shard(key)
	if !kv.checkShard(key) {
		return &yqkvpb.Reply{Status: yqkvpb.ERR_WRONG_GROUP}
	}
	val, err := kv.Shards[shard].Get(key)
	if err != nil {
		return &yqkvpb.Reply{Status: yqkvpb.ERR_NO_KEY}
	}
	return &yqkvpb.Reply{Status: yqkvpb.OK, Value: val}
}

func (kv *ShardKV) applyPut(key, value string) *yqkvpb.Reply {
	shard := client.Key2shard(key)
	if !kv.checkShard(key) {
		return &yqkvpb.Reply{Status: yqkvpb.ERR_WRONG_GROUP}
	}
	_ = kv.Shards[shard].Put(key, value)
	return &yqkvpb.Reply{Status: yqkvpb.OK}
}

func (kv *ShardKV) applyAppend(key, value string) *yqkvpb.Reply {
	shard := client.Key2shard(key)
	if !kv.checkShard(key) {
		return &yqkvpb.Reply{Status: yqkvpb.ERR_WRONG_GROUP}
	}
	_ = kv.Shards[shard].Append(key, value)
	return &yqkvpb.Reply{Status: yqkvpb.OK}
}

func (kv *ShardKV) applyDeleteShards(configNum int64, shardIDs []int64) *yqkvpb.Reply {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if configNum != kv.CurrentConfig.Num {
		return &yqkvpb.Reply{Status: yqkvpb.OK}
	}

	for _, shardID := range shardIDs {
		if kv.Shards[shardID].Status() == yqkvpb.BE_PULLING {
			kv.Shards[shardID] = NewShard(&yqkvpb.Shard{ID: shardID, Status: yqkvpb.SERVING})
		} else if kv.Shards[shardID].Status() == yqkvpb.GC {
			kv.Shards[shardID].SetStatus(yqkvpb.SERVING)
		}
	}
	return &yqkvpb.Reply{Status: yqkvpb.OK}
}

func (kv *ShardKV) checkShard(key string) bool {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	shardID := client.Key2shard(key)
	shard, ok := kv.Shards[shardID]
	if !ok {
		return false
	}
	ok = (shard.Status() == yqkvpb.SERVING || shard.Status() == yqkvpb.GC) && kv.CurrentConfig.Shards[shardID] == kv.gid
	if !ok {
		//kv.logger.Printf("WRONG GROUP shardID=%v status=%v at gid=%v configNum=%v", shardID, shard.Status, kv.CurrentConfig.Shards[shardID], kv.CurrentConfig.Num)
	}
	return ok
}

// Kill
// the tester calls Kill() when a ShardKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *ShardKV) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.
	close(kv.stop)
}

func (kv *ShardKV) shouldApply(request sessionpb.SessionHeader) bool {
	return !kv.SessionMap.HaveProcessed(request)
}

func (kv *ShardKV) applyReconfigure(nextConfig *shardctlpb.Config) *yqkvpb.Reply {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if nextConfig.Num == kv.CurrentConfig.Num+1 {
		kv.updateShardsStatus(nextConfig)
		kv.LastConfig = kv.CurrentConfig
		kv.CurrentConfig = shardctl.NewConfig(nextConfig)
		return &yqkvpb.Reply{Status: yqkvpb.OK}
	}
	return &yqkvpb.Reply{Status: yqkvpb.ERR_OUT_DATED_CONFIG}
}

func (kv *ShardKV) updateShardsStatus(nextConfig *shardctlpb.Config) {
	if nextConfig.Num <= 1 {
		return
	}
	old := kv.CurrentConfig
	for shardID, currentGid := range nextConfig.Shards {
		oldGid := old.Shards[shardID]
		if oldGid == kv.gid && currentGid != kv.gid {
			kv.Shards[int64(shardID)].SetStatus(yqkvpb.BE_PULLING)
		} else if oldGid != kv.gid && currentGid == kv.gid {
			kv.Shards[int64(shardID)].SetStatus(yqkvpb.PULLING)
		}
	}
	return
}

func (kv *ShardKV) applyInsertShards(response *yqkvpb.PullShardsResponse) *yqkvpb.Reply {
	if response.ConfigNum < kv.CurrentConfig.Num {
		return &yqkvpb.Reply{Status: yqkvpb.ERR_OUT_DATED_CONFIG}
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	for _, shard := range response.Shards {
		if kv.Shards[shard.ID].Status() == yqkvpb.PULLING {
			kv.Shards[shard.ID] = NewShard(shard)
			kv.Shards[shard.ID].SetStatus(yqkvpb.GC)
		}
	}
	kv.SessionMap.Merge(response.ServerSessions)
	return &yqkvpb.Reply{Status: yqkvpb.OK}
}

func (kv *ShardKV) shouldSnapshot() bool {
	return kv.maxraftstate > 0 && kv.rf.LogSize() > kv.maxraftstate
}

func (kv *ShardKV) createSnapshot() {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(*kv.StateMachine)
	if err != nil {
		kv.logger.Panic(err)
	}
	snapshot := buffer.Bytes()
	index := atomic.LoadUint64(&kv.AppliedIndex)
	kv.rf.Snapshot(index, snapshot)
	//kv.logger.Printf("CREATE SNAPSHOT index=%v", index)
}

func (kv *ShardKV) installSnapshot(snapshot []byte) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	buffer := bytes.NewBuffer(snapshot)
	dec := gob.NewDecoder(buffer)
	var state StateMachine
	err := dec.Decode(&state)
	if err != nil {
		return
	}
	kv.StateMachine = &state
	kv.SessionMap.ResetLock()
	kv.waitApply.Trigger(state.AppliedIndex)
	kv.logger.Printf("UPDATE STATE MACHINE FROM SNAPSHOT")
}

func (kv *ShardKV) applyEmpty() *yqkvpb.Reply {
	return &yqkvpb.Reply{Status: yqkvpb.OK}
}

// StartServer
// servers[] contains the ports of the servers in this group.
//
// me is the index of the current server in servers[].
//
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
//
// the k/v server should snapshot when Raft's saved state exceeds
// maxraftstate bytes, in order to allow Raft to garbage-collect its
// log. if maxraftstate is -1, you don't need to snapshot.
//
// gid is this group's GID, for interacting with the shardctl.
//
// pass ctrlers[] to shardctl.MakeClerk() so you can send
// RPCs to the shardctl.
//
// make_end(servername) turns a server name from a
// NextConfig.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs. You'll need this to send RPCs to other groups.
//
// look at client.go for examples of how to use ctrlers[]
// and make_end() to send RPCs to the group owning a specific shard.
//
// StartServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartServer(serverAddr []string, me uint64, persister *raft.Persister, maxraftstate int64, gid int64, ctrlers []shardctlpb.ShardctlClient, makeEnd func(string) yqkvpb.ShardKVClient) (*ShardKV, error) {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	gob.Register(shardctl.Config{})

	kv := new(ShardKV)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.make_end = makeEnd
	kv.gid = gid
	kv.ctrlers = ctrlers
	kv.mu = &sync.RWMutex{}
	// Your initialization code here.

	// Use something like this to talk to the shardctl:
	// kv.mck = shardctl.MakeClerk(kv.ctrlers)

	kv.applyCh = make(chan raft.ApplyMsg)
	rf, err := raft.Make(serverAddr, me, persister, kv.applyCh)
	if err != nil {
		return nil, fmt.Errorf("make shardKV error:[%w]", err)
	}
	kv.rf = rf
	kv.StateMachine = &StateMachine{
		Shards:        make(map[int64]*Shard),
		SessionMap:    session.NewSessionMap(),
		AppliedIndex:  0,
		LastConfig:    nil,
		CurrentConfig: &shardctl.Config{Num: 0},
	}
	kv.waitApply = wait.NewTimeList()
	kv.controller = shardctl.MakeClerk(kv.ctrlers)
	kv.clerk = client.MakeClerk(ctrlers, makeEnd)
	kv.logger = log.New(os.Stdout, fmt.Sprintf("[SHARD KV SERVER %v-%v] ", kv.gid, kv.me), log.LstdFlags|log.Lmsgprefix)
	kv.stop = make(chan struct{})
	if persister.SnapshotSize() > 0 {
		kv.installSnapshot(persister.ReadSnapshot())
	}
	go kv.run()
	go kv.Monitor(kv.configureAction, 100*time.Millisecond)
	go kv.Monitor(kv.migratingAction, 100*time.Millisecond)
	go kv.Monitor(kv.gcAction, 100*time.Millisecond)
	go kv.Monitor(kv.checkEntryInCurrentTermAction, 100*time.Millisecond)
	return kv, nil
}
