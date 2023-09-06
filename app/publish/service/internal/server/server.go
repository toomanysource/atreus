package server

import (
	"context"

	"github.com/toomanysource/atreus/app/publish/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
	stdgrpc "google.golang.org/grpc"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewUserClient, NewFavoriteClient)

type (
	UserConn     stdgrpc.ClientConnInterface
	FavoriteConn stdgrpc.ClientConnInterface
)

// NewUserClient 创建一个User服务客户端，接收User服务数据
func NewUserClient(c *conf.Client, logger log.Logger) UserConn {
	logs := log.NewHelper(log.With(logger, "module", "server/user"))
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.User.To),
		grpc.WithMiddleware(
			recovery.Recovery(),
			logging.Client(logger),
		),
	)
	if err != nil {
		logs.Fatalf("user service connect error, %v", err)
	}
	logs.Info("user service connect successfully")
	return conn
}

// NewFavoriteClient 创建一个Favorite服务客户端，接收Favorite服务数据
func NewFavoriteClient(c *conf.Client, logger log.Logger) FavoriteConn {
	logs := log.NewHelper(log.With(logger, "module", "server/favorite"))
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Favorite.To),
		grpc.WithMiddleware(
			recovery.Recovery(),
			logging.Client(logger),
		),
	)
	if err != nil {
		logs.Fatalf("favorite service connect error, %v", err)
	}
	logs.Info("favorite service connect successfully")
	return conn
}
