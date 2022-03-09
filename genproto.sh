protoc -I=. -I=$GOPATH/pkg/mod -I=$GOPATH/pkg/mod/github.com/gogo/protobuf@v1.3.2/gogoproto --gofast_out=plugins=grpc,paths=source_relative:. ./raft/raftpb/raft.proto
protoc -I=. -I=$GOPATH/pkg/mod -I=$GOPATH/pkg/mod/github.com/gogo/protobuf@v1.3.2/gogoproto --gofast_out=plugins=grpc,paths=source_relative:. ./session/sessionpb/session.proto
protoc -I=. -I=$GOPATH/pkg/mod -I=$GOPATH/pkg/mod/github.com/gogo/protobuf@v1.3.2/gogoproto -I=./session/sessionpb --gofast_out=plugins=grpc,paths=source_relative:. ./shardctl/shardctlpb/shardctl.proto
