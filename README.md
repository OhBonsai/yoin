# yoin
Complete Bitcoin Wallet And Client Implement by Golang

完整的比特币实现，代码里面有很多中文注释。详情可以看[博客]()

## 代码结构
- core 区块链核心代码
  - addr.go 地址相关
  - block.go 区块相关
  - blockdb.go 区块数据库
  - chain.go 链相关
  - chain_tree.go 区块链树模型，提供一些跳到区块的方法
  - tx.go 交易相关
  - uin256.go hash32byte一些辅助函数
  - util.go 一些功能函数
  - dbif.go 数据库接口
  －
