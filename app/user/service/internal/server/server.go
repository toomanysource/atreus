package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/hashicorp/consul/api"

	relationv1 "github.com/toomanysource/atreus/api/relation/service/v1"
	"github.com/toomanysource/atreus/app/user/service/internal/conf"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewRelationClient, NewDiscovery, NewRegistrar)

// NewRelationClient 创建一个Relation服务客户端，接收Relation服务数据
func NewRelationClient(r registry.Discovery, logger log.Logger) relationv1.RelationServiceClient {
	logs := log.NewHelper(log.With(logger, "module", "server/relation"))
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///atreus.relation.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
			logging.Client(logger),
		),
	)
	if err != nil {
		logs.Fatalf("Relation service connect error, %v", err)
	}
	logs.Info("user service connect successfully")
	return relationv1.NewRelationServiceClient(conn)
}

func NewDiscovery(conf *conf.Registry) registry.Discovery {
	c := api.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	cli, err := api.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(false))
	return r
}

func NewRegistrar(conf *conf.Registry) registry.Registrar {
	c := api.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	cli, err := api.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(false))
	return r
}
