package session

import (
	"errors"
	"github.com/gogo/protobuf/types"
	pb "github.com/wangyuqi0706/yqkv/session/sessionpb"

	//"google.golang.org/protobuf/proto"
	"github.com/gogo/protobuf/proto"
	"log"
	"sync"
	"sync/atomic"
)

var ErrDuplicateSession error = errors.New(pb.ERR_DUPLICATE.String())
var ErrReturned error = errors.New(pb.ERR_RETURNED.String())

type Response any

type SessionMap struct {
	Map map[int64]*ServerSession
	mu  *sync.RWMutex
}

func (s *SessionMap) getSession(id int64) *ServerSession {
	session, ok := s.Map[id]
	if !ok {
		return nil
	}
	return session
}

func (s *SessionMap) Lock() {
	s.mu.Lock()
}

func (s *SessionMap) Unlock() {
	s.mu.Unlock()
}

func (s *SessionMap) CreateSession(id int64) (newSession *ServerSession, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, loaded := s.Map[id]
	if loaded {
		return nil, ErrDuplicateSession
	}
	s.Map[id] = NewServerSession(id)
	return s.Map[id], nil
}

func (s *SessionMap) SetSession(id int64, session *ServerSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Map[id] = session
}

func (s *SessionMap) HasSession(id int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Map[id]
	return ok
}

func (s *SessionMap) IsDuplicate(request *pb.SessionHeader) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session := s.getSession(request.ID)
	if session == nil {
		return false
	}
	if request.Seq <= session.ProcessedSeq() {
		return true
	}
	return false
}

func (s *SessionMap) GetResponse(request *pb.SessionHeader) (proto.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getSession(request.ID).getResponse(request.Seq)
}

func (s *SessionMap) SetResponse(request *pb.SessionHeader, resp proto.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getSession(request.ID).setResponse(request.Seq, resp)
}

// ResetLock Calling after decoding to reset a Mutex/
// Should be called before any other methods
func (s *SessionMap) ResetLock() {
	s.mu = &sync.RWMutex{}
}

func (s *SessionMap) HaveProcessed(request pb.SessionHeader) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.HasSession(request.ID) {
		return false
	}
	session := s.getSession(request.ID)
	return request.Seq <= session.ProcessedSeq()
}

func (s *SessionMap) Merge(from map[int64]*pb.ServerSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for clientID, session := range from {
		localSession := s.getSession(clientID)
		if localSession == nil || localSession.ProcessedSeq() < session.ProcessedSeq {
			s.Map[clientID] = NewServerSessionFrom(session)
		}
	}
}

func (s *SessionMap) ProtoMessage() map[int64]*pb.ServerSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pm := make(map[int64]*pb.ServerSession)
	for id, session := range s.Map {
		pm[id] = session.ProtoMessage()
	}
	return pm
}

func (s *SessionMap) DeepCopy() *SessionMap {
	nm := make(map[int64]*ServerSession)
	for id, session := range s.Map {
		nm[id] = session.deepCopy()
	}
	ns := &SessionMap{
		Map: nm,
		mu:  &sync.RWMutex{},
	}
	return ns
}

type ServerSession struct {
	session *pb.ServerSession
}

func (s *ServerSession) deepCopy() *ServerSession {
	clone := proto.Clone(s.session).(*pb.ServerSession)
	return NewServerSessionFrom(clone)
}

func (s *ServerSession) ProtoMessage() *pb.ServerSession {
	clone := proto.Clone(s.session).(*pb.ServerSession)
	return clone
}

func (s *ServerSession) getResponse(num uint64) (proto.Message, error) {
	if num != s.session.ProcessedSeq {
		return nil, ErrReturned
	}
	var resp proto.Message
	err := types.UnmarshalAny(s.session.LastResponse, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *ServerSession) setResponse(num uint64, resp proto.Message) error {
	if num < s.session.ProcessedSeq {
		log.Panicf("should not set a processed request. num=%v processedSeq=%v ", num, s.session.ProcessedSeq)
	}
	s.session.ProcessedSeq = num
	last, err := types.MarshalAny(resp)
	s.session.LastResponse = last
	if err != nil {
		return err
	}
	return nil
}

func (s *ServerSession) ProcessedSeq() uint64 {
	return s.session.ProcessedSeq
}

func NewServerSession(id int64) *ServerSession {
	return &ServerSession{
		session: &pb.ServerSession{
			ID:           id,
			ProcessedSeq: 0,
			LastResponse: nil,
		},
	}
}

func NewServerSessionFrom(session *pb.ServerSession) *ServerSession {
	return &ServerSession{
		session: session,
	}
}

type ClientSession struct {
	int64        int64
	ProcessedSeq uint64
	NextSeq      uint64
}

func NewClientSession(id int64) *ClientSession {
	return &ClientSession{
		int64:        id,
		ProcessedSeq: 0,
		NextSeq:      0,
	}
}

func (c *ClientSession) NextHeader() *pb.SessionHeader {
	return &pb.SessionHeader{ID: c.int64, Seq: atomic.AddUint64(&c.NextSeq, 1)}
}

func NewSessionMap() *SessionMap {
	sm := SessionMap{
		Map: make(map[int64]*ServerSession),
		mu:  &sync.RWMutex{},
	}
	return &sm
}
