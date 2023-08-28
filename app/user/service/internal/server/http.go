package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "github.com/toomanysource/atreus/api/user/service/v1"
	"github.com/toomanysource/atreus/app/user/service/internal/conf"
	"github.com/toomanysource/atreus/app/user/service/internal/service"
)

// NewHTTPServer new a user service HTTP server.
func NewHTTPServer(c *conf.Server, user *service.UserService, logger log.Logger) *http.Server {
	opts := []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterUserServiceHTTPServer(srv, user)
	return srv
}
