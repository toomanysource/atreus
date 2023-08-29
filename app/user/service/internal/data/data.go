package data

import (
	"context"

	"gorm.io/gorm/logger"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/app/user/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGormDb, NewRedisConn, NewUserRepo)

var (
	maxOpenConnection = 100
	maxIdleConnecton  = 10
)

// Data .
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
	log *log.Helper
}

// NewData .
func NewData(db *gorm.DB, rdb *redis.Client, logger log.Logger) (*Data, func(), error) {
	logs := log.NewHelper(log.With(logger, "resources", "data"))
	cleanup := func() {
		logs.Info("closing the data resources")
		if err := rdb.Close(); err != nil {
			logs.Errorf("close redis connection failed: %w", err)
			return
		}
		logs.Infof("close redis connection successfully")
	}
	data := &Data{
		db:  db,
		rdb: rdb,
		log: log.NewHelper(logger),
	}
	return data, cleanup, nil
}

// NewGormDb .
func NewGormDb(c *conf.Data, l log.Logger) *gorm.DB {
	logs := log.NewHelper(log.With(l, "resources", "data/mysql"))
	dsn := c.Database.Source
	open, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		logs.Fatalf("database connect failed, error: %v", err.Error())
	}
	db, _ := open.DB()
	// 连接池配置
	db.SetMaxOpenConns(maxOpenConnection)
	db.SetMaxIdleConns(maxIdleConnecton)
	return open
}

// NewRedisConn .
func NewRedisConn(c *conf.Data, l log.Logger) *redis.Client {
	logs := log.NewHelper(log.With(l, "resources", "data/redis"))
	client := redis.NewClient(&redis.Options{
		DB:           int(c.Redis.Db),
		Addr:         c.Redis.Addr,
		Username:     c.Redis.Username,
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		Password:     c.Redis.Password,
	})
	// ping Redis客户端
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		logs.Fatalf("ping redis failure, err: %v", err)
	}
	logs.Info("redis run successfully!")
	return client
}
