package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
	stdgrpc "google.golang.org/grpc"

	"github.com/toomanysource/atreus/app/user/service/internal/conf"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewRelationClient)

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
