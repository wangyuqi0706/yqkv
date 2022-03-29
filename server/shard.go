package server

import (
	"errors"
	"github.com/gogo/protobuf/proto"
	"github.com/wangyuqi0706/yqkv/yqkvpb"
)

type KV interface {
	Get(string) (string, error)
	Put(string, string) error
	Append(string, string) error
	Empty() bool
}

type Shard struct {
	shard *yqkvpb.Shard
}

func NewShard(shard *yqkvpb.Shard) *Shard {
	return &Shard{shard: proto.Clone(shard).(*yqkvpb.Shard)}
}

func (s *Shard) Get(key string) (string, error) {
	if value, ok := s.shard.KV[key]; ok {
		return value, nil
	}
	return "", errors.New(yqkvpb.ERR_NO_KEY.String())
}

func (s *Shard) Put(key string, value string) error {
	s.shard.KV[key] = value
	return nil
}

func (s *Shard) Append(key string, value string) error {
	s.shard.KV[key] += value
	return nil
}

func (s *Shard) Empty() bool {
	return len(s.shard.KV) == 0
}

func (s *Shard) Copy() *Shard {
	return &Shard{proto.Clone(s.shard).(*yqkvpb.Shard)}
}

func (s *Shard) ProtoMessage() *yqkvpb.Shard {
	return proto.Clone(s.shard).(*yqkvpb.Shard)
}

func (s *Shard) Status() yqkvpb.ShardStatus {
	return s.shard.Status
}

func (s *Shard) ID() int64 {
	return s.shard.ID
}

func (s *Shard) SetStatus(status yqkvpb.ShardStatus) {
	s.shard.Status = status
}
