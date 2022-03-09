package shardctl

import (
	"fmt"
	"github.com/wangyuqi0706/yqkv/raft"
	"github.com/wangyuqi0706/yqkv/raft/raftpb"
	"github.com/wangyuqi0706/yqkv/session"
	pb "github.com/wangyuqi0706/yqkv/shardctl/shardctlpb"
	"go.etcd.io/etcd/pkg/v3/wait"
	"log"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type ShardCtl struct {
	mu      *sync.Mutex
	me      uint64
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	// Your data here.
	waitApply  wait.WaitTime
	SessionMap *session.SessionMap

	AppliedIndex uint64

	clientSession *session.ClientSession
	configs       []*Config // indexed by config num
	stop          chan struct {
	}

	logger *log.Logger
}

type OpType string

const (
	ErrWrongLeader string = "ErrWrongLeader"
	OK             string = "OK"
)

func (sc *ShardCtl) Execute(args *pb.Args, reply *pb.Reply) {
	if sc.SessionMap.IsDuplicate(args.SessionHeader) {
		if resp, err := sc.SessionMap.GetResponse(args.SessionHeader); err != nil {
			reply.Status = err.Error()
		} else {
			*reply = *resp.(*pb.Reply)
		}
		sc.logger.Printf("DUPLICATE %v request=%v reply=%v", args.Type, args.SessionHeader, reply)
		return
	}
	data, _ := args.Marshal()
	index, _, isLead := sc.rf.Start(data)
	if !isLead {
		reply.WrongLeader = true
		reply.Status = ErrWrongLeader
		return
	}
	<-sc.waitApply.Wait(index)
	resp, err := sc.SessionMap.GetResponse(args.SessionHeader)
	if err != nil {
		reply.Status = err.Error()
		return
	}
	*reply = *resp.(*pb.Reply)
	//sc.logger.Printf("RETURN %v index=%v request=%v reply=%v", args.Type, index, args.Request, reply)
}

func (sc *ShardCtl) run() {
	for {
		select {
		case msg := <-sc.applyCh:
			if msg.CommandValid && msg.Command != nil {
				var reply *pb.Reply
				var err error
				var args pb.Args
				_ = args.Unmarshal(msg.Command)
				if sc.SessionMap.IsDuplicate(args.SessionHeader) {
					atomic.StoreUint64(&sc.AppliedIndex, msg.CommandIndex)
					sc.waitApply.Trigger(msg.CommandIndex)
					continue
				}
				switch args.Type {
				case pb.JOIN:
					reply, err = sc.applyJoin(args.NewGroups)
				case pb.LEAVE:
					reply, err = sc.applyLeave(args.LeaveGIDs)
				case pb.MOVE:
					reply, err = sc.applyMove(args.MoveGID, args.MoveShard)
				case pb.QUERY:
					reply, err = sc.applyQuery(args.QueryNum)
				case pb.EMPTY:
					reply = &pb.Reply{Status: OK, WrongLeader: false}
					err = nil
				}
				if err != nil {
					continue
				}
				if !sc.SessionMap.HasSession(args.SessionHeader.ID) {
					_, _ = sc.SessionMap.CreateSession(args.SessionHeader.ID)
				}
				_ = sc.SessionMap.SetResponse(args.SessionHeader, reply)
				atomic.StoreUint64(&sc.AppliedIndex, msg.CommandIndex)
				sc.waitApply.Trigger(msg.CommandIndex)
			}
		case <-sc.stop:
			return
		}
	}
}

// Kill
// the tester calls Kill() when a ShardCtl instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (sc *ShardCtl) Kill() {
	sc.rf.Kill()
	// Your code here, if desired.
	close(sc.stop)
}

func (sc *ShardCtl) applyJoin(groups []*pb.Group) (*pb.Reply, error) {
	cfg := sc.newConfig()
	// add new groups
	//join(cfg, groups)
	cfg.Join(groups)
	// map shard -> gid
	cfg.Reshard()
	sc.configs = append(sc.configs, cfg)
	return &pb.Reply{Status: OK}, nil
}

func (sc *ShardCtl) applyLeave(gids []int64) (*pb.Reply, error) {
	cfg := sc.newConfig()
	// delete some gid
	//leave(cfg, gids)
	// re-shard
	cfg.Leave(gids)
	cfg.Reshard()
	sc.configs = append(sc.configs, cfg)
	return &pb.Reply{WrongLeader: false, Status: OK}, nil
}

func (sc *ShardCtl) applyMove(gid int64, shard int64) (*pb.Reply, error) {
	cfg := sc.newConfig()
	cfg.Shards[shard] = gid
	sc.configs = append(sc.configs, cfg)
	return &pb.Reply{WrongLeader: false, Status: OK}, nil
}

func (sc *ShardCtl) applyQuery(queryNum int64) (*pb.Reply, error) {
	var resp session.Response
	if queryNum == -1 || queryNum >= int64(len(sc.configs)) {
		resp = sc.configs[len(sc.configs)-1]
	} else {
		resp = sc.configs[queryNum]
	}
	return &pb.Reply{Status: OK, Config: resp.(*pb.Config)}, nil
}

func (sc *ShardCtl) newConfig() *Config {
	oldCfg := sc.configs[len(sc.configs)-1]
	ncfg := &Config{
		Num:    oldCfg.Num + 1,
		Shards: oldCfg.Shards,
		Groups: map[int64]*pb.Group{},
	}

	for _, group := range oldCfg.Groups {
		ncfg.Groups[group.ID] = group
	}
	return ncfg
}

func (sc *ShardCtl) checkEntryInCurrentTermAction() {
	if !sc.rf.HasCurrentTermLog() {
		args := &pb.Args{Type: pb.EMPTY, SessionHeader: sc.clientSession.NextHeader()}
		sc.Execute(args, &pb.Reply{})
		return
	}
}

func (sc *ShardCtl) Monitor(action func(), timeout time.Duration) {
	for {
		select {
		case <-sc.stop:
			return
		default:
			if _, isLeader := sc.rf.GetState(); isLeader {
				action()
			}
			time.Sleep(timeout)
		}
	}
}

// StartServer
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant shardctl service.
// "me" is the index of the current server in servers[].
//
func StartServer(servers []raftpb.RaftRPCClient, me uint64, persister *raft.Persister) *ShardCtl {
	sc := new(ShardCtl)
	sc.me = me

	sc.configs = make([]*Config, 1)
	sc.configs[0].Groups = map[int64]*pb.Group{}

	sc.applyCh = make(chan raft.ApplyMsg)
	sc.rf = raft.Make(servers, me, persister, sc.applyCh)

	// Your code here.
	sc.SessionMap = session.NewSessionMap()
	sc.clientSession = session.NewClientSession(rand.Int63())
	sc.waitApply = wait.NewTimeList()
	sc.mu = &sync.Mutex{}
	sc.AppliedIndex = 0
	sc.stop = make(chan struct{})
	sc.logger = log.New(os.Stdout, fmt.Sprintf("[CTL SERVER %v] ", sc.me), log.LstdFlags|log.Lmsgprefix)

	go sc.run()
	go sc.Monitor(sc.checkEntryInCurrentTermAction, 100*time.Millisecond)

	return sc
}
