#kitex -module raft  -type protobuf -service raftpb raft.proto
protoc -I=. -I=$GOPATH/pkg/mod -I=$GOPATH/pkg/mod/github.com/gogo/protobuf@v1.3.2/gogoproto --gofast_out=plugins=grpc:. raft.proto
