package server

import (
	"context"

	"github.com/toomanysource/atreus/app/favorite/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
	stdgrpc "google.golang.org/grpc"
)

var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewPublishClient)

type (
	PublishConn stdgrpc.ClientConnInterface
)

// NewPublishClient 创建一个Publish服务客户端，接收Publish服务数据
func NewPublishClient(c *conf.Client, logger log.Logger) PublishConn {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Publish.To),
		grpc.WithMiddleware(
			recovery.Recovery(),
			logging.Client(logger),
		),
	)
	if err != nil {
		log.Fatalf("Error connecting to Publish Services, err : %v", err)
	}
	return conn
}
