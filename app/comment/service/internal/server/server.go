package server

import (
	"context"

	"github.com/toomanysource/atreus/app/comment/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/hashicorp/consul/api"

	userv1 "github.com/toomanysource/atreus/api/user/service/v1"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewUserClient, NewDiscovery, NewRegistrar)

// NewUserClient 创建一个User服务客户端，接收User服务数据
func NewUserClient(r registry.Discovery, logger log.Logger) userv1.UserServiceClient {
	logs := log.NewHelper(log.With(logger, "module", "server/user"))
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///atreus.user.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
			logging.Client(logger),
		),
	)
	if err != nil {
		logs.Fatalf("user service connect error, %v", err)
	}
	logs.Info("user service connect successfully")
	return userv1.NewUserServiceClient(conn)
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
