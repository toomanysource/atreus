# Atreus

[中文](README_cn.md) | [English](README.md)

**Atreus** 是一个基于 **Kratos** 微服务框架，实现高并发、高性能的短视频应用服务端。

- **高性能**: 用高速缓存 Redis 提高数据访问的速度和性能。还采用了 Minio 实现了毫秒级的上传存储。
- **高并发**: 使用 Kafka 作为高效异步消息处理，提高系统吞吐和稳定性。

## 项目结构

Atreus 参照了[Kratos Layout](https://github.com/go-kratos/kratos-layout). 这种设计理念基于 **DDD**.
![](docs/img/readme/atreus-project-structure.png)

```
❯ tree -L 1
.
├── LICENSE
├── README.md
├── README_cn.md
├── _data           //  保存所有服务和组件的数据.
├── api             // `.proto` API 文件和生成的 `pb.go` 文件.
├── app             // 服务实现
├── configs         // docker 配置文件
├── docker
├── third_party     // api 依赖的第三方 proto
├── pkg             // 第三方包和通用包
├── middleware      // 自定义中间件
├── docs
├── Makefile
├── make
├── go.mod
└── go.sum
```

**App** 结构

```
.
├── cmd             // 服务入口文件
│   ├── main.go
│   ├── wire.go     // 使用 wire 维护依赖注入
│   └── wire_gen.go
├── configs         // 本地调试的配置文件.
└── internal        // 业务逻辑代码.
    ├── biz         // 业务逻辑的组装层.
    ├── conf        // config 的结构定义，使用 proto 生成
    ├── data        // 业务数据访问.
    ├── server      // http和grpc实例的创建和配置.
    └── service     // 实现了api定义的服务层.
```

## 技术栈

- [Kratos](https://github.com/go-kratos/kratos)
- [MySQL](https://www.mysql.com/)
- [GORM](https://github.com/go-gorm/gorm)
- [Redis](https://github.com/redis/go-redis)
- [Kafka](https://github.com/segmentio/kafka-go)
- [Minio](https://github.com/minio/minio)

## 开始

我们在 Docker 构建项目，你只需要运行如下命令:

```
make docker-compose-up
```
>  注意! 你需要将`/configs/service/publish/config.yaml`里的 minio `endpointExtra` 字段改成自己本机IP.
## 如何贡献

更多详情，请访问[contribute](./docs/contribute).

## 贡献者

- [alilestera](https://github.com/alilestera)
- [intyouss](https://github.com/intyouss)
- [mo3et](https://github.com/mo3et)
- [Dyamidsteve](https://github.com/Dyamidsteve)
- [meguminkin](https://github.com/meguminkin)
- [FirwoodLin](https://github.com/FirwoodLin)
- [Li1Mo0N](https://github.com/Li1Mo0N)

## 许可证

Atreus is open-sourced software licensed under the [Apache License 2.0](./LICENSE).