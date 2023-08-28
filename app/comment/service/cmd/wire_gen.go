// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/toomanysource/atreus/app/comment/service/internal/biz"
	"github.com/toomanysource/atreus/app/comment/service/internal/conf"
	"github.com/toomanysource/atreus/app/comment/service/internal/data"
	"github.com/toomanysource/atreus/app/comment/service/internal/server"
	"github.com/toomanysource/atreus/app/comment/service/internal/service"

	_ "go.uber.org/automaxprocs"
)

// Injectors from wire.go:

// wireApp init kratos application.
func wireApp(confServer *conf.Server, client *conf.Client, confData *conf.Data, jwt *conf.JWT, logger log.Logger) (*kratos.App, func(), error) {
	db := data.NewMysqlConn(confData, logger)
	redisClient := data.NewRedisConn(confData, logger)
	dataData, cleanup, err := data.NewData(db, redisClient, logger)
	if err != nil {
		return nil, nil, err
	}
	userConn := server.NewUserClient(client, logger)
	publishConn := server.NewPublishClient(client, logger)
	commentRepo := data.NewCommentRepo(dataData, userConn, publishConn, logger)
	commentUsecase := biz.NewCommentUsecase(jwt, commentRepo, logger)
	commentService := service.NewCommentService(commentUsecase, logger)
	grpcServer := server.NewGRPCServer(confServer, commentService, logger)
	httpServer := server.NewHTTPServer(confServer, jwt, commentService, logger)
	app := newApp(logger, grpcServer, httpServer)
	return app, func() {
		cleanup()
	}, nil
}
