package shardctl

//
// Shardctrler clerk.
//

import (
	"context"
	"github.com/wangyuqi0706/yqkv/session"
	pb "github.com/wangyuqi0706/yqkv/shardctl/shardctlpb"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

type ClientConfig struct {
	RequestTimeout time.Duration
	RetryIntervals []time.Duration

	// Retry infinitely when Retries <0
	Retries int
}

const RequestTimeout = 5 * 200 * time.Millisecond

func defaultConfig() ClientConfig {
	return ClientConfig{RequestTimeout, []time.Duration{50 * time.Millisecond}, -1}
}

type Clerk struct {
	servers []pb.ShardctlClient
	// Your data here.
	session      *session.ClientSession
	cachedLeader int32
	clientConfig ClientConfig
	mu           *sync.Mutex

	logger *log.Logger
}

func MakeClerk(servers []pb.ShardctlClient) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	// Your code here.
	ck.cachedLeader = -1
	ck.logger = log.New(os.Stdout, "[CTL CLIENT] ", log.LstdFlags|log.Lmsgprefix)
	ck.clientConfig = defaultConfig()
	ck.session = session.NewClientSession(rand.Int63())
	ck.mu = &sync.Mutex{}
	return ck
}

func (ck *Clerk) Query(num int64) *pb.Config {
	args := &pb.Args{
		Type:     pb.QUERY,
		QueryNum: num,
	}
	reply := &pb.Reply{}
	// Your code here.
	ck.call(args, reply)
	return reply.Config
}

func (ck *Clerk) Join(servers []*pb.Group) {
	args := &pb.Args{
		Type:      pb.JOIN,
		NewGroups: servers,
	}
	reply := &pb.Reply{}

	// Your code here.
	ck.call(args, reply)

}

func (ck *Clerk) Leave(gids []int64) {
	args := &pb.Args{
		Type:      pb.LEAVE,
		LeaveGIDs: gids,
	}
	reply := &pb.Reply{}
	// Your code here.

	ck.call(args, reply)

}

func (ck *Clerk) Move(shard int64, gid int64) {
	args := &pb.Args{
		Type:      pb.MOVE,
		MoveShard: shard,
		MoveGID:   gid,
	}
	reply := &pb.Reply{}
	// Your code here.
	ck.call(args, reply)
}

func (ck *Clerk) call(args *pb.Args, reply *pb.Reply) {
	ck.mu.Lock()
	defer ck.mu.Unlock()
	retried := 0
	args.SessionHeader = ck.session.NextHeader()
	for ck.clientConfig.Retries < 0 || retried < ck.clientConfig.Retries {
		servers := ck.servers
		okChan := make(chan bool)
		to := rand.Intn(len(servers))
		r := &pb.Reply{}
		server := servers[to]
		go func() {
			var err error
			r, err = server.Execute(context.TODO(), args)
			if err != nil {
				okChan <- false
			} else {
				okChan <- true
			}
		}()
		select {
		case ok := <-okChan:
			if ok {
				switch r.Status {
				// return
				case pb.OK:
					*reply = *r
					//ck.logger.Printf("REQUEST SUCCESS args=%v reply=%v", args, reply)
					return
				case pb.ERR_WRONG_LEADER:
				}
			}
			// Else, Retry
		case <-time.After(ck.clientConfig.RequestTimeout * 3):
			//
			ck.logger.Printf("TIMEOUT args=%v", args)
		}
	}

	//ck.logger.Printf("SEND REQUEST: to=%v arg=%v", to, args)
	var interval time.Duration
	if ck.clientConfig.Retries < 0 {
		interval = ck.clientConfig.RetryIntervals[0]
	} else {
		interval = ck.clientConfig.RetryIntervals[retried]
	}
	retried++
	time.Sleep(interval)
}
