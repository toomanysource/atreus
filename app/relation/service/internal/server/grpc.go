package server

import (
	v1 "github.com/toomanysource/atreus/api/relation/service/v1"
	"github.com/toomanysource/atreus/app/relation/service/internal/conf"
	"github.com/toomanysource/atreus/app/relation/service/internal/service"

	"github.com/go-kratos/kratos/v2/middleware/logging"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a relation service gRPC server.
func NewGRPCServer(c *conf.Server, relation *service.RelationService, logger log.Logger) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterRelationServiceServer(srv, relation)
	return srv
}
