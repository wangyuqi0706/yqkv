package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, Term, isleader)
//   start agreement on a new log entry
// rf.GetState() (Term, isLeader)
//   ask a Raft for its current Term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"math/rand"
	"os"
	"time"

	//	"bytes"
	"sync"
	"sync/atomic"

	pb "github.com/wangyuqi0706/yqkv/raft/raftpb"
)

// ApplyMsg
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      []byte
	CommandIndex uint64

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  uint64
	SnapshotIndex uint64
}

type ServerState int32

const (
	ServerStateLeader    ServerState = 1
	ServerStateFollower  ServerState = 2
	ServerStateCandidate ServerState = 3
)

var (
	ErrTimeout = errTimeout()
	ErrNoReply = errNoReply()
)

func errTimeout() error { return errors.New("request timeout") }
func errNoReply() error { return errors.New("request no reply") }

const HeartBeatInterval = 100 * time.Millisecond
const EmptyEntryValue = "Empty"

type HardState struct {
	currentTerm int
	votedFor    int
	log         *raftLog
}

const None uint64 = 0

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        *sync.RWMutex      // Lock to protect shared access to this peer's state
	peers     []pb.RaftRPCClient // RPC end points of all peers
	persister *Persister         // Object to hold this peer's persisted state
	me        uint64             // this peer's index into peers[]
	dead      int32              // set by Kill()

	// Your Entries here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	state                ServerState
	votes                uint64
	heartBeatReceiveChan chan bool
	// cancel the election ticker
	cancelTicker   context.CancelFunc
	cancelElection context.CancelFunc

	// Persistent state on all servers:
	pb.RaftState
	log raftLog

	// Volatile state on all servers:
	lastApplied  uint64
	newEntryCond *sync.Cond

	//Volatile state on leaders:
	nextIndex    []uint64
	matchIndex   []uint64
	cancelLeader context.CancelFunc

	// snapshot metadata
	applyChan chan ApplyMsg

	leaderId uint64
	logger   *log.Logger
	done     chan struct{}
}

type PersistentState struct {
	CurrentTerm int
	VotedFor    int
	Log         *raftLog
}

// GetState return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (uint64, bool) {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	// Your code here (2A).
	return rf.HardState.Term, rf.state == ServerStateLeader
}

func (rf *Raft) GetLeaderId() uint64 {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.leaderId
}

func (rf *Raft) LogSize() int64 {
	return rf.persister.RaftStateSize()
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	data := rf.persistRaftStateData()
	rf.persister.SaveRaftState(data)
	//log.Printf("SAVE PERSIST:server=%v currentTerm=%v,votedFor=%v,log=%v", rf.me, p.CurrentTerm, p.VotedFor, p.Log)

}

// persistRaftStateData
// Decode Persistent States to []byte
func (rf *Raft) persistRaftStateData() []byte {
	data, err := rf.RaftState.Marshal()
	if err != nil {
		rf.logger.Printf("ENCODE ERROR: %v", err)
		return nil
	}
	return data
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersistRaftState(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		rf.HardState.Term = 0
		rf.HardState.VotedFor = 0
		rf.Log = &pb.Log{Entries: make([]*pb.Entry, 16)}
		rf.persist()
		return
	}
	var p pb.RaftState

	err := p.Unmarshal(data)
	if err != nil {
		rf.logger.Printf("DECODE ERR: %v", p)
	}
	rf.RaftState = p
	rf.lastApplied = p.Log.IncludedIndex
	rf.logger.Printf("READ PERSIST:server=%v currentTerm=%v,votedFor=%v,log=%v", rf.me, rf.HardState.Term, rf.HardState.Term, rf.Log)
}

// CondInstallSnapshot
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm uint64, lastIncludedIndex uint64, snapshot []byte) bool {
	// Your code here (2D).
	rf.logger.Printf("CondInstallSnapshot:server=%v ,lastIncludedTerm=%v lastIncludedIndex=%v", rf.me, lastIncludedTerm, lastIncludedIndex)
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// Refuse if it is an old Snapshot
	if rf.log.LastIncludedTerm() > lastIncludedTerm {
		return false
	}
	if rf.log.LastIncludedTerm() == lastIncludedTerm && rf.log.LastIncludedIndex() >= lastIncludedIndex {
		return false
	}

	rf.lastApplied = lastIncludedIndex
	// Trim log
	rf.log.CompactForSnapshot(lastIncludedIndex, lastIncludedTerm)
	// Persist state
	stateData := rf.persistRaftStateData()
	rf.persister.SaveStateAndSnapshot(stateData, snapshot)
	return true
}

// Snapshot
// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index uint64, snapshot []byte) {
	// Your code here (2D).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.HardState.Commit = max(rf.HardState.Commit, index)
	rf.lastApplied = max(rf.lastApplied, index)
	// Trim log if snapshot is older than current
	if index <= rf.log.LastEntry().Index {
		rf.log.CompactForSnapshot(index, rf.log.EntryAt(index).Term)
	}
	// Persist state
	stateData := rf.persistRaftStateData()
	rf.persister.SaveStateAndSnapshot(stateData, snapshot)
	//rf.logger.Printf("SNAPSHOT:server=%v ,index=%v,log=%v", rf.me, index, rf.log)
}

// RequestVoteArgs
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your Entries here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// newRequestVoteArgs
// Return a new RequestVoteArgs according to the Raft object
func (rf *Raft) newRequestVoteArgs() *pb.RequestVoteArgs {
	rf.mu.RLock()
	defer rf.mu.RUnlock()

	args := &pb.RequestVoteArgs{
		Args: &pb.Args{
			Term: rf.getTerm(),
		},
		CandidateID:  rf.me,
		LastLogIndex: rf.log.LastIncludedIndex(),
		LastLogTerm:  rf.log.LastIncludedTerm(),
	}
	return args
}

// setRejectRequestVoteReply
// Let the reply representing reject vote
func (rf *Raft) setRejectRequestVoteReply(reply *pb.RequestVoteReply) {
	reply.Term = rf.getTerm()
	reply.VoteGranted = false
}

func (rf *Raft) callRequestVote(ctx context.Context, done chan *pb.RequestVoteReply, server int) {
	args := rf.newRequestVoteArgs()
	//log.Printf("SEND REQUEST VOTE:From %v to %v Term=%v", rf.me, server,args.Term)
	//okChan := make(chan bool)
	rf.mu.RLock()
	if rf.state != ServerStateCandidate {
		rf.mu.RUnlock()
		return
	}
	rf.mu.RUnlock()
	reply, err := rf.peers[server].RequestVote(ctx, args)
	select {
	case <-ctx.Done():
		return
	default:
		if err != nil {
			rf.mu.Lock()
			//log.Printf("GET VOTE:From %v to candidate=%v at Term %v  VoteGranted=%v", server, rf.me, reply.Term, reply.VoteGranted)

			if reply.Term > rf.getTerm() {
				rf.convertToFollower(reply.Term)
				rf.mu.Unlock()
				return
			}
			done <- reply
			rf.mu.Unlock()
		}
	}
}

func (rf *Raft) newAppendEntriesArgs(server int, isHeartBeat bool) *pb.AppendEntriesArgs {
	args := &pb.AppendEntriesArgs{
		Args: &pb.Args{
			Term:   rf.getTerm(),
			Leader: rf.me,
		},
		PrevLogIndex: rf.nextIndex[server] - 1,
		Entries:      nil,
		LeaderCommit: rf.getCommit(),
	}

	if args.PrevLogIndex == rf.log.LastIncludedIndex() {
		args.PrevLogTerm = rf.log.LastIncludedTerm()
	} else {
		args.PrevLogTerm = rf.log.EntryAt(rf.nextIndex[server] - 1).Term
	}

	if rf.getCommit() <= rf.log.LastIncludedIndex() {
		args.CommitTerm = rf.log.LastIncludedTerm()
	} else {
		args.CommitTerm = rf.log.EntryAt(rf.getCommit()).Term
	}

	if !isHeartBeat {
		args.Entries = rf.log.EntriesAfter(rf.nextIndex[server])
	}
	return args
}

func (rf *Raft) callAppendEntries(ctx context.Context, server int, isHeartBeat bool) error {
	rf.mu.RLock()
	if rf.state != ServerStateLeader {
		rf.mu.RUnlock()
		return fmt.Errorf("NOT LEADER")
	}

	// If server's Snapshot is older than leader, call InstallSnapshot
	if rf.nextIndex[server] <= rf.log.LastIncludedIndex() {
		rf.mu.RUnlock()
		go rf.callInstallSnapshot(ctx, server)
		return fmt.Errorf("OLD SNAPSHOT")
	}
	args := rf.newAppendEntriesArgs(server, isHeartBeat)
	rf.mu.RUnlock()

	reply := &pb.AppendEntriesReply{}
	okChan := make(chan bool)
	timeoutCtx, cancel := context.WithTimeout(ctx, HeartBeatInterval*2)
	defer cancel()
	//Call RPC with goroutine
	go func() {
		//log.Printf("SEND APPEND ENTRY:leader=%v,server=%v,entries=%v", rf.me, server, args.Entries)
		var err error
		reply, err = rf.peers[server].AppendEntries(ctx, args)
		if err != nil {
			okChan <- false
		}
		okChan <- true
	}()
	select {
	// RPC Returns
	case ok := <-okChan:
		if !ok {
			return ErrNoReply
		}
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if reply.Term > rf.getTerm() {
			rf.convertToFollower(reply.Term)
			return fmt.Errorf("BECOME FOLLOWER")
		}
		if rf.state != ServerStateLeader {
			return fmt.Errorf("NOT LEADER")
		}

		if reply.Status == pb.Success {
			if len(args.Entries) > 0 {
				lastEntry := args.Entries[len(args.Entries)-1]
				rf.nextIndex[server] = lastEntry.Index + 1
				rf.matchIndex[server] = lastEntry.Index
				rf.checkIfEntryShouldBeCommitted(lastEntry.Index)
			}
		} else {
			if reply.Status == pb.NotContain {
				// Follower's log is shorter
				//oldNextIndex := rf.nextIndex[server]
				//log.Printf("NOT CONTAIN:server=%v,reply.XTerm=%v,XIndex=%v,XLen=%v", server, reply.XTerm, reply.XIndex, reply.XLen)
				if reply.XTerm == 0 && reply.XLen != 0 {
					rf.nextIndex[server] = reply.XLen
					//log.Printf("Follower's log is shorter:server=%v,prevIndex=%v,oldNextIndex=%v, newNextIndex=%v", server, args.PrevLogIndex, oldNextIndex, rf.nextIndex[server])
					return fmt.Errorf("NOT CONTAIN")
				}

				// Search Xterm in log
				entry := rf.log.FirstEntryAtTerm(reply.XTerm)

				if entry != nil {
					// Last entry with XTerm
					lastEntry := rf.log.LastEntryAtTerm(reply.XTerm)
					if lastEntry == nil {
						rf.nextIndex[server]--
					} else {
						rf.nextIndex[server] = lastEntry.Index
					}
					//log.Printf("Last entry with XTerm: prevIndex=%v,oldNextIndex=%v, newNextIndex=%v", args.PrevLogIndex, oldNextIndex, rf.nextIndex[server])
				} else {
					// Doesn't find XTerm
					rf.nextIndex[server] = reply.XIndex
					//log.Printf("Doesn't find XTerm: prevIndex=%v,oldNextIndex=%v, newNextIndex=%v", args.PrevLogIndex, oldNextIndex, rf.nextIndex[server])
				}
			}
		}

	// RPC call be canceled
	case <-timeoutCtx.Done():
		//log.Printf("CALL APPEND ENTRIES TIMEOUT")
		return ErrTimeout
	}
	return nil
}

// Start
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the First return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command []byte) (index uint64, term uint64, isLeader bool) {
	index = 0
	term = 0
	// Your code here (2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	isLeader = rf.state == ServerStateLeader
	if !isLeader {
		return 0, 0, false
	}

	index = rf.nextIndex[rf.me]
	rf.nextIndex[rf.me]++
	term = rf.getTerm()
	//rf.logger.Printf("START: server=%v index=%v command=%v", rf.me, index, command)
	entry := pb.Entry{
		Term:  term,
		Index: index,
		Data:  command,
	}
	rf.appendEntryToLog(entry)
	return index, term, true
}

func (rf *Raft) HasCurrentTermLog() bool {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.log.LastEntry().Term >= rf.getTerm()
}

// sync call
func (rf *Raft) replicateToAllServers(ctx context.Context) {
	for i := 0; i < len(rf.peers); i++ {
		if uint64(i) == rf.me {
			continue
		}
		// start a goroutine to replicate log for each server
		go rf.replicateLogToServer(ctx, i)
	}
}

// replicateLogToServer
// Starting replicate logs to a server with timeout
func (rf *Raft) replicateLogToServer(ctx context.Context, server int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			rf.mu.Lock()
			rf.newEntryCond.Wait()
			rf.mu.Unlock()
			_ = rf.callAppendEntries(ctx, server, false)
		}
	}
}

func (rf *Raft) checkIfEntryShouldBeCommitted(n uint64) {
	// If index n has committed
	if rf.getCommit() >= n {
		return
	}
	// Raft never commits log entries from previous terms by counting replicas.
	if rf.log.EntryAt(n).Term < rf.getTerm() {
		return
	}
	count := 0
	for _, matchIndex := range rf.matchIndex {
		if matchIndex >= n {
			count++
		}
	}
	if count > len(rf.peers)/2 {
		rf.commitEntryBeforeIndex(n)
	}
}

func (rf *Raft) checkQuorum() bool {
	rf.logger.Printf("checkQuorum")
	return rf.broadcastBeat() > len(rf.peers)
}

func (rf *Raft) broadcastBeat() int {
	count := int32(1)
	wg := sync.WaitGroup{}
	for i := range rf.peers {
		if uint64(i) != rf.me {
			wg.Add(1)
			go func() {
				err := rf.callAppendEntries(context.Background(), i, true)
				wg.Done()
				if !errors.Is(err, ErrNoReply) && !errors.Is(err, ErrTimeout) {
					atomic.AddInt32(&count, 1)
				}
			}()
		}
	}
	wg.Wait()
	rf.logger.Printf("broadcastBeat: count=%v", count)
	return int(count)
}

func (rf *Raft) commitEntryBeforeIndex(index uint64) {
	if index <= rf.getCommit() {
		//rf.logger.Printf("ENTRY HAS COMMITTED:server=%v index=%v", rf.me, index)
		return
	}
	//rf.logger.Printf("COMMIT ENTRY: server=%v index=%v cmd=%v", rf.me, index, rf.log.EntryAt(index).Value)
	rf.HardState.Commit = index
}

// Kill
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
	close(rf.done)
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker(term uint64) {
	rf.mu.Lock()
	if rf.cancelTicker != nil {
		//log.Printf("CANCEL TICKER:New ticker server=%v", rf.me)
		rf.cancelTicker()
	}
	ctx, cancel := context.WithCancel(context.Background())
	rf.cancelTicker = cancel
	rf.mu.Unlock()

	lowBound := 2 * HeartBeatInterval
	// timeout is a random number which is in [5 * HeartBeatInterval,10 * HeartBeatInterval]
	timeout := lowBound + time.Duration(rand.Int63n(int64(3*lowBound)))
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for !rf.killed() {
		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().
		select {
		case <-timer.C:
			go rf.launchElection(term + 1)
			return
		case <-rf.heartBeatReceiveChan:
			timer.Reset(timeout)
			//log.Printf("Heart Beating server=%v Term=%v", rf.me, Term)
			continue
		case <-ctx.Done():
			return
		}

	}
}

// RPC handlers

func (rf *Raft) AppendEntries(ctx context.Context, args *pb.AppendEntriesArgs) (reply *pb.AppendEntriesReply, err error) {
	rf.heartBeatReceiveChan <- true
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Entries != nil {
		//log.Printf("RECEIVE APPEND ENTRY:leader=%v,server=%v,entries=%v", args.LeaderId, rf.me, args.Entries)
	} else {
		//log.Printf("RECEIVE HEARTBEAT:leader=%v,server=%v", args.LeaderId, rf.me)
	}
	reply.Term = rf.getTerm()
	// 1. Reply false if Term < currentTerm (§5.1)
	if args.Args.Term < rf.getTerm() {
		reply.Status = pb.OldTerm
		return
	}

	if args.Args.Term > rf.getTerm() {
		rf.convertToFollower(args.Args.Term)
	}

	rf.leaderId = args.Args.Leader

	if args.PrevLogIndex < rf.log.IncludedIndex {
		reply.Status = pb.OldSnapshot
		rf.logger.Printf("AppendEntries: Entry has been trimmed args=%v", args)
		return
	}

	// 2. Reply false if log does not contain an entry at prevLogIndex whose Term matches prevLogTerm (§5.3)
	if args.PrevLogIndex > rf.log.Len() || rf.log.EntryAt(args.PrevLogIndex).Term != args.PrevLogTerm {
		rf.logger.Printf("APPEND ENTRIES NOT CONTAIN:PrevLogIndex=%v Term=%v len=%v log=%v", args.PrevLogIndex, args.PrevLogTerm, rf.log.Len(), rf.log)
		reply.Status = pb.NotContain
		if args.PrevLogIndex > rf.log.Len() {
			reply.XLen = rf.log.LastEntry().Index + 1
			return
		}

		prevEntry := rf.log.EntryAt(args.PrevLogIndex)
		reply.XTerm = prevEntry.Term
		fe := rf.log.FirstEntryAtTerm(prevEntry.Term)
		reply.XIndex = fe.Index
		return
	}

	// 3. If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and
	// all that follow it (§5.3)
	for _, entry := range args.Entries {
		if entry.Index < rf.log.LastIncludedIndex() {
			reply.Status = pb.OldSnapshot
			rf.logger.Printf("AppendEntries: Entry has been trimmed args=%v", args)
			return
		}
		if entry.Index <= rf.log.Len() && entry.Term != rf.log.EntryAt(entry.Index).Term {
			rf.log.DiscardAfterAnd(entry.Index)
			rf.persist()
			//log.Printf("DELETE CONFLICTED ENTRIES:server=%v,entries=%v", rf.me, deleted)
			break
		}
	}

	// 4. Append any new entries not already in the log
	for _, entry := range args.Entries {
		rf.appendEntryToLog(*entry)
	}

	// 5. If leaderCommit > Commit, set Commit = min(leaderCommit, index of last new entry)
	// If a heart-beating package with args.LeaderCommit=n arrives before an uncommitted entry at log[n] is deleted, log[n]
	// could be committed by accident.
	if args.LeaderCommit > rf.getCommit() {
		commitIndex := min(args.LeaderCommit, rf.log.LastEntry().Index)
		if rf.log.EntryAt(commitIndex).Term != args.CommitTerm {
			return
		}
		rf.commitEntryBeforeIndex(commitIndex)
	} else {
		//log.Printf("args.LeaderCommit{%v} <= rf.Commit{%v} server=%v", args.LeaderCommit, rf.Commit, rf.me)
	}

	reply.Status = pb.Success
	return reply, err
}

func (rf *Raft) RequestVote(ctx context.Context, args *pb.RequestVoteArgs) (reply *pb.RequestVoteReply, err error) {
	// Your code here (2A, 2B).
	//log.Printf("RECEIVE RPC: RequestVote, args = %v", args)
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// Reject request if argument's Term less than current Term
	if args.Args.Term < rf.getTerm() {
		rf.logger.Printf("REJECT VOTE:candidate=%v, server=%v, arg.Term = %v,currentTerm=%v", args.CandidateID, rf.me, args.Args.Term, rf.getTerm())
		rf.setRejectRequestVoteReply(reply)
		return
	}

	// Become a follower when args.Term > rf.currentTerm
	if args.Args.Term >= rf.getTerm() {
		rf.convertToFollower(args.Args.Term)
	}

	// If votedFor is null or candidateId,
	// and candidate’s log is at least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
	if (rf.getVotedFor() < 0 || args.CandidateID == rf.getVotedFor()) && rf.log.notUpToDate(args.LastLogIndex, args.LastLogTerm) {
		rf.HardState.VotedFor = args.CandidateID
		rf.persist()
		reply.VoteGranted = true
		reply.Term = args.Args.Term
		//log.Printf("SERVER %v VOTED TO: %v ", rf.me, args.CandidateId)
		return
	}
	rf.setRejectRequestVoteReply(reply)
	return reply, nil
}

func (rf *Raft) InstallSnapshot(ctx context.Context, args *pb.InstallSnapshotArgs) (reply *pb.InstallSnapshotReply, err error) {
	rf.logger.Printf("INSTALL SNAPSHOT RPC from=%v includedIndex=%v", args.Args.Leader, args.Snapshot.Metadata.Index)
	rf.mu.RLock()
	reply.Term = rf.getTerm()
	if args.Args.Term < rf.getTerm() {
		rf.mu.RUnlock()
		return
	}

	if args.Args.Term > rf.getTerm() {
		rf.convertToFollower(args.Args.Term)
	}
	rf.leaderId = args.Args.Leader
	msg := ApplyMsg{
		CommandValid:  false,
		Command:       nil,
		CommandIndex:  0,
		SnapshotValid: true,
		Snapshot:      args.Snapshot.Data,
		SnapshotTerm:  args.Snapshot.Metadata.Term,
		SnapshotIndex: args.Snapshot.Metadata.Index,
	}
	rf.mu.RUnlock()
	rf.applyChan <- msg
	return reply, nil
}

func (rf *Raft) newInstallSnapshotArgs() *pb.InstallSnapshotArgs {
	snapshot := rf.persister.ReadSnapshot()
	return &pb.InstallSnapshotArgs{
		Args: &pb.Args{
			Term:   rf.getTerm(),
			Leader: rf.me,
		},
		Snapshot: &pb.Snapshot{
			Metadata: &pb.SnapshotMetaData{
				Index: rf.log.LastIncludedIndex(),
				Term:  rf.log.LastIncludedIndex(),
			},
			Data: snapshot,
		},
	}
}

func (rf *Raft) callInstallSnapshot(ctx context.Context, server int) {
	//rf.logger.Printf("CALL InstallSnapshot:leader=%v,server=%v", rf.me, server)
	rf.mu.RLock()
	args := rf.newInstallSnapshotArgs()
	rf.mu.RUnlock()
	reply := &pb.InstallSnapshotReply{}

	okChan := make(chan bool)
	go func() {
		var err error
		reply, err = rf.peers[server].InstallSnapshot(ctx, args)
		if err != nil {
			okChan <- false
			return
		}
		okChan <- true
	}()

	select {
	case ok := <-okChan:
		if ok {
			rf.mu.Lock()
			if rf.getTerm() < reply.Term {
				rf.convertToFollower(reply.Term)
			} else {
				rf.nextIndex[server] = args.Snapshot.Metadata.Index + 1
				rf.matchIndex[server] = args.Snapshot.Metadata.Index
			}
			rf.mu.Unlock()
		}
	case <-ctx.Done():
		return
	}
}

// State Conversion

func (rf *Raft) convertToLeader() {
	rf.logger.Printf("LEADER ELECTED: leader=%v, term=%v ", rf.me, rf.getTerm())
	if rf.cancelLeader != nil {
		rf.cancelLeader()
		rf.cancelLeader = nil
	}
	if rf.cancelTicker != nil {
		rf.cancelTicker()
		rf.cancelTicker = nil
		//log.Printf("CANCEL TICKER:Convert to leader server=%v", rf.me)
	}
	rf.state = ServerStateLeader
	rf.leaderId = rf.me
	rf.nextIndex = make([]uint64, len(rf.peers))
	rf.matchIndex = make([]uint64, len(rf.peers))
	rf.HardState.VotedFor = None
	rf.persist()

	//Append a new entry when elected to make sure all entries that created at old term will be committed by the new leader.

	for i := 0; i < len(rf.peers); i++ {
		rf.nextIndex[i] = rf.log.LastEntry().Index + 1
		rf.matchIndex[i] = 0
	}

	leaderCtx, cancel := context.WithCancel(context.Background())
	rf.cancelLeader = cancel
	go rf.sendHeartBeatPeriodicallyToAll(leaderCtx)
	go rf.replicateToAllServers(leaderCtx)
}

func (rf *Raft) applyCommittedLogs() {
	for {
		rf.mu.Lock()
		if rf.lastApplied < rf.getCommit() {
			rf.lastApplied++
			entry := rf.log.EntryAt(rf.lastApplied)
			rf.mu.Unlock()
			msg := ApplyMsg{
				CommandValid:  true,
				Command:       entry.Data,
				CommandIndex:  entry.Index,
				SnapshotValid: false,
				Snapshot:      nil,
				SnapshotTerm:  0,
				SnapshotIndex: 0,
			}
			rf.applyChan <- msg
			//rf.logger.Printf("APPLY ENTRY: server=%v index=%v cmd=%v", rf.me, entry.Index, entry.Value)
		} else {
			rf.mu.Unlock()
			time.Sleep(10 * time.Millisecond)
		}
	}

}

func (rf *Raft) sendHeartBeatPeriodicallyToAll(ctx context.Context) {
	me := rf.me
	timer := time.NewTimer(HeartBeatInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			go func() {
				for i := 0; i < len(rf.peers); i++ {
					if uint64(i) != me {
						go func(i int) {
							_ = rf.callAppendEntries(ctx, i, true)
						}(i)
					}
				}
				//if int(okCount) <= len(rf.peers)/2 {
				//	rf.mu.Lock()
				//	//rf.logger.Printf("GIVE UP LEADER")
				//	//rf.convertToFollower(rf.currentTerm)
				//	rf.mu.Unlock()
				//	return
				//}
			}()
			rf.newEntryCond.Broadcast()
			timer.Reset(HeartBeatInterval)
		case <-ctx.Done():
			return
		}

	}
}

func (rf *Raft) convertToFollower(term uint64) {
	//log.Printf("CONVERT TO FOLLOWER: server=%v oldTerm=%v newTerm=%v", rf.me, rf.currentTerm, Term)
	if rf.cancelLeader != nil {
		rf.cancelLeader()
	}
	rf.HardState.Term = term
	rf.state = ServerStateFollower
	rf.votes = 0
	rf.leaderId = 0
	rf.HardState.VotedFor = None
	rf.persist()
	go rf.ticker(term)
}

func (rf *Raft) launchElection(newTerm uint64) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.state == ServerStateLeader {
		rf.cancelLeader()
	}

	ctx, cancel := context.WithCancel(context.Background())
	if rf.cancelElection != nil {
		rf.cancelElection()
	}
	rf.cancelElection = cancel
	// convert to candidate
	rf.state = ServerStateCandidate
	rf.leaderId = 0
	// increment Term
	rf.HardState.Term = newTerm
	//rf.logger.Printf("LAUNCH ELECTION: server=%v newTerm=%v", rf.me, rf.currentTerm)

	// vote for self
	rf.votes = 1
	rf.HardState.VotedFor = rf.me
	rf.persist()

	go rf.sendRequestVoteToAll(ctx)
	go rf.ticker(newTerm)
}

func (rf *Raft) sendRequestVoteToAll(ctx context.Context) {
	replyChan := make(chan *pb.RequestVoteReply, len(rf.peers))
	// Send request vote to all servers
	for i := range rf.peers {
		if uint64(i) == rf.me {
			continue
		}
		go rf.callRequestVote(ctx, replyChan, i)
	}
	for {
		select {
		case reply := <-replyChan:
			rf.mu.Lock()
			//log.Printf("RECEIVE VOTE")
			if reply.VoteGranted {
				rf.votes++
			}
			// Elected when gain majority votes
			if rf.votes > uint64(len(rf.peers)/2) {
				rf.convertToLeader()
				rf.mu.Unlock()
				return
			}
			rf.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (rf *Raft) appendEntryToLog(entry pb.Entry) {
	if entry.Index == rf.log.Len()+1 {
		rf.log.Append(entry)
		rf.persist()
		//log.Printf("APPEND ENTRY: server=%v,index=%v,cmd=%v", rf.me, entry.Index, entry.Value)
		if rf.state == ServerStateLeader {
			rf.matchIndex[rf.me] = entry.Index
		}
		rf.newEntryCond.Broadcast()
	}
}

func (rf *Raft) getTerm() uint64 {
	return rf.HardState.Term
}

func (rf *Raft) getVotedFor() uint64 {
	return rf.HardState.VotedFor
}

func (rf *Raft) getCommit() uint64 {
	return rf.HardState.Commit
}

// Make
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peersAddr []string, me uint64,
	persister *Persister, applyCh chan ApplyMsg) (*Raft, error) {
	rf := &Raft{}
	rf.logger = log.New(os.Stdout, fmt.Sprintf("[RAFT %v] ", me), log.LstdFlags|log.Lmsgprefix)
	peers, err := makePeers(peersAddr)
	if err != nil {
		return nil, fmt.Errorf("make raft error:[%w]", err)
	}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.mu = new(sync.RWMutex)

	rf.state = ServerStateFollower
	rf.votes = 0
	rf.heartBeatReceiveChan = make(chan bool)
	rf.cancelTicker = nil
	rf.cancelElection = nil

	rf.newEntryCond = sync.NewCond(rf.mu)
	rf.HardState.Commit = 0
	rf.lastApplied = 0

	rf.leaderId = 0

	rf.applyChan = applyCh
	rf.done = make(chan struct{})
	// initialize from state persisted before a crash
	rf.readPersistRaftState(persister.ReadRaftState())
	// start ticker goroutine to start elections
	rf.convertToFollower(rf.getTerm())

	go rf.applyCommittedLogs()

	return rf, nil
}

func makePeers(addrs []string) ([]pb.RaftRPCClient, error) {
	clients := make([]pb.RaftRPCClient, 0, len(addrs))
	for _, addr := range addrs {
		conn, err := grpc.Dial(addr)
		if err != nil {
			return nil, err
		}
		client := pb.NewRaftRPCClient(conn)
		clients = append(clients, client)
	}
	return clients, nil
}
