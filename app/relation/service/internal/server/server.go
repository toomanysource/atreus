package server

import (
	"context"

	"github.com/toomanysource/atreus/app/relation/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
	stdgrpc "google.golang.org/grpc"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewUserClient)

// NewUserClient 创建一个User服务客户端，接收User服务数据
func NewUserClient(c *conf.Client, logger log.Logger) *stdgrpc.ClientConn {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.User.To),
		grpc.WithMiddleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
	)
	if err != nil {
		log.Fatalf("Error connecting to User Services, err : %v", err)
	}
	return conn
}
