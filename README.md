# yqkv
![language](https://img.shields.io/github/go-mod/go-version/wangyuqi0706/yqkv?style=flat-square)
![GitHub](https://img.shields.io/github/license/wangyuqi0706/yqkv?style=flat-square)

线性一致性的分布式key-value存储系统

# 特性
- 线性一致性
- 支持横向扩展
- 分片热迁移

# 上手指南
```
client := shardkv.MakeClient(...)
client.Put(key,"hello")
client.Append(key," world")
v := client.Get(key) // v = "hello world"
```


# 安装要求
1. Go >= 1.17  [官方网站](https://go.dev/)
2. Protocol Buffers 3.14.0 [官方网站](https://developers.google.com/protocol-buffers)


# 开源许可
该项目签署了GPLv3 授权许可，详情请参阅LICENSE.md