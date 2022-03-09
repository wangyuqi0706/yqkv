package session

import (
	"errors"
	pb "github.com/wangyuqi0706/yqkv/session/sessionpb"
	"log"
	"sync"
	"sync/atomic"
)

var ErrDuplicateSession error = errors.New("ErrDuplicateSession")

const ErrReturned = "ErrReturned"

type Response interface{}

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
	_, dup := session.ResponseMap[request.Seq]
	return dup
}

func (s *SessionMap) GetResponse(request *pb.SessionHeader) (Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getSession(request.ID).getResponse(request.Seq)
}

func (s *SessionMap) SetResponse(request *pb.SessionHeader, resp Response) error {
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
	return request.Seq <= session.ProcessedSeq
}

func (s *SessionMap) Merge(from *SessionMap) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for clientID, session := range from.Map {
		localSession := s.getSession(clientID)
		if localSession == nil || localSession.ProcessedSeq < session.ProcessedSeq {
			s.Map[clientID] = session.deepCopy()
		}
	}
}

func (s *SessionMap) DeepCopy() *SessionMap {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sm := NewSessionMap()
	for clientID, session := range s.Map {
		sm.Map[clientID] = session.deepCopy()
	}
	return sm
}

type ServerSession struct {
	int64        int64
	ProcessedSeq uint64
	ResponseMap  map[uint64]Response
	LastResponse Response
}

func (s *ServerSession) deepCopy() *ServerSession {
	copied := NewServerSession(s.int64)
	copied.ProcessedSeq = s.ProcessedSeq
	copied.ResponseMap = make(map[uint64]Response)
	for sequenceNum, response := range s.ResponseMap {
		copied.ResponseMap[sequenceNum] = response
	}
	copied.LastResponse = s.LastResponse
	return copied
}

func (s *ServerSession) DiscardBefore(num uint64) {
	for sequenceNum := range s.ResponseMap {
		if sequenceNum < num {
			s.ResponseMap[sequenceNum] = nil
			delete(s.ResponseMap, sequenceNum)
		}
	}
}

func (s *ServerSession) getResponse(num uint64) (Response, error) {
	resp, ok := s.ResponseMap[num]
	if !ok && num <= s.ProcessedSeq {
		return nil, errors.New(ErrReturned)
	}
	//s.DiscardBefore(num)
	return resp, nil
}

func (s *ServerSession) setResponse(num uint64, resp Response) error {
	if num < s.ProcessedSeq {
		log.Panicf("should not set a processed request. num=%v processedSeq=%v ", num, s.ProcessedSeq)
	}
	s.ResponseMap[num] = resp
	s.ProcessedSeq = num
	return nil
}

func NewServerSession(id int64) *ServerSession {
	return &ServerSession{
		int64:        id,
		ProcessedSeq: 0,
		ResponseMap:  make(map[uint64]Response),
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

func (c *ClientSession) receiveResponse(num uint64) {
	addr := &c.ProcessedSeq
	for {
		old := atomic.LoadUint64(addr)
		swapped := atomic.CompareAndSwapUint64(addr, old, MinSequenceNum(num, old))
		if swapped {
			break
		}
	}

}

func MinSequenceNum(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func NewSessionMap() *SessionMap {
	sm := SessionMap{
		Map: make(map[int64]*ServerSession),
		mu:  &sync.RWMutex{},
	}
	return &sm
}
