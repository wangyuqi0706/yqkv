package shardctl

import (
	"github.com/lafikl/consistent"
	pb "github.com/wangyuqi0706/yqkv/shardctl/shardctlpb"
	"strconv"
)

const NShards = 1024

type Config struct {
	Num    int64
	Shards [NShards]int64
	Groups map[int64]*pb.Group
}

func (cfg *Config) Gids() []int64 {
	gids := make([]int64, 0, len(cfg.Groups))
	for id, _ := range cfg.Groups {
		gids = append(gids, id)
	}
	return gids
}

func (cfg *Config) Join(newGroups []*pb.Group) {
	for _, group := range newGroups {
		if _, ok := cfg.Groups[group.ID]; !ok {
			cfg.Groups[group.ID] = group
		}
	}
}

func (cfg *Config) Leave(gids []int64) {
	for _, gid := range gids {
		cfg.Groups[gid] = nil
		delete(cfg.Groups, gid)
	}
}

func (cfg *Config) Reshard() {
	if len(cfg.Groups) <= 0 {
		cfg.Shards = [NShards]int64{}
		return
	}

	c := consistent.New()
	for _, gid := range cfg.Gids() {
		c.Add(strconv.FormatInt(gid, 10))
	}
	for shard := range cfg.Shards {
		gidstring, _ := c.GetLeast(strconv.Itoa(shard))
		c.Inc(gidstring)
		cfg.Shards[shard], _ = strconv.ParseInt(gidstring, 10, 64)
	}
}
