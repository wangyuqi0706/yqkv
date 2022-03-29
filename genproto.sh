#!/bin/bash
protoc -I=. -I=$GOPATH/pkg/mod -I=$GOPATH/pkg/mod/github.com/gogo/protobuf/gogoproto --gofast_out=plugins=grpc,paths=source_relative:. ./raft/raftpb/raft.proto
protoc -I=. -I=$GOPATH/pkg/mod -I=$GOPATH/pkg/mod/github.com/gogo/protobuf/gogoproto --gofast_out=plugins=grpc,Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,paths=source_relative:. ./session/sessionpb/session.proto
protoc -I=. -I=$GOPATH/pkg/mod -I=$GOPATH/pkg/mod/github.com/gogo/protobuf/gogoproto -I=./session/sessionpb --gofast_out=plugins=grpc,paths=source_relative:. ./shardctl/shardctlpb/shardctl.proto
protoc -I=. -I=$GOPATH/pkg/mod -I=$GOPATH/pkg/mod/github.com/gogo/protobuf/gogoproto -I=./session/sessionpb -I=./shardctl/shardctlpb  --gofast_out=plugins=grpc,paths=source_relative:. ./yqkvpb/yqkvpb.proto
