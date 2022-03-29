module github.com/wangyuqi0706/yqkv

go 1.18

replace (
	github.com/wangyuqi0706/yqkv/raft => ./raft
	github.com/wangyuqi0706/yqkv/session => ./session
	github.com/wangyuqi0706/yqkv/shardctl => ./shardctl
)

require (
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/wangyuqi0706/yqkv/raft v0.0.0
	github.com/wangyuqi0706/yqkv/session v0.0.0-00010101000000-000000000000
	github.com/wangyuqi0706/yqkv/shardctl v0.0.0-00010101000000-000000000000
	go.etcd.io/etcd/pkg/v3 v3.5.2
	google.golang.org/grpc v1.44.0
)

require (
	github.com/lafikl/consistent v0.0.0-20210222184039-5e8acd7e59f2 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
)
