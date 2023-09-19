package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"

	consul "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	consulAPI "github.com/hashicorp/consul/api"
	stdgrpc "google.golang.org/grpc"

	"github.com/toomanysource/atreus/app/user/service/internal/conf"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewRelationClient, NewDiscovery, NewRegistrar)

type RelationConn stdgrpc.ClientConnInterface

// NewRelationClient 创建一个Relation服务客户端，接收Relation服务数据
func NewRelationClient(c *conf.Client, logger log.Logger) RelationConn {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Relation.To),
		grpc.WithMiddleware(
			recovery.Recovery(),
			logging.Client(logger),
		),
	)
	if err != nil {
		log.Fatalf("Error connecting to Relation Services, err : %v", err)
	}
	log.Info("Relation Services connected.")
	return conn
}

func NewDiscovery(conf *conf.Registry) registry.Discovery {
	c := consulAPI.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	cli, err := consulAPI.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(false))
	return r
}

func NewRegistrar(conf *conf.Registry) registry.Registrar {
	c := consulAPI.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	cli, err := consulAPI.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(false))
	return r
}
