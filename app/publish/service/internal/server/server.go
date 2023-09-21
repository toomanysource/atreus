package server

import (
	"context"

	"github.com/toomanysource/atreus/app/publish/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/hashicorp/consul/api"

	favoritev1 "github.com/toomanysource/atreus/api/favorite/service/v1"
	userv1 "github.com/toomanysource/atreus/api/user/service/v1"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewUserClient, NewFavoriteClient, NewDiscovery, NewRegistrar)

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

// NewFavoriteClient 创建一个Favorite服务客户端，接收Favorite服务数据
func NewFavoriteClient(r registry.Discovery, logger log.Logger) favoritev1.FavoriteServiceClient {
	logs := log.NewHelper(log.With(logger, "module", "server/favorite"))
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///atreus.favorite.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
			logging.Client(logger),
		),
	)
	if err != nil {
		logs.Fatalf("favorite service connect error, %v", err)
	}
	logs.Info("favorite service connect successfully")
	return favoritev1.NewFavoriteServiceClient(conn)
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
