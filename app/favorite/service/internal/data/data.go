package data

import (
	"context"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/app/favorite/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ProviderSet = wire.NewSet(NewData, NewFavoriteRepo, NewUserRepo, NewPublishRepo, NewMysqlConn, NewRedisConn)

type Data struct {
	db    *gorm.DB
	cache *redis.Client
	log   *log.Helper
}

func NewData(db *gorm.DB, cache *redis.Client, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "data/favorite"))
	// 并发关闭所有数据库连接，后期根据Redis与Mysql是否数据同步修改
	// MySQL 会自动关闭连接，但是 Redis 不会，所以需要手动关闭
	cleanup := func() {
		_, err := cache.Ping(context.Background()).Result()
		if err != nil {
			return
		}
		if err = cache.Close(); err != nil {
			logHelper.Errorf("Redis connection closure failed, err: %w", err)
		}
		logHelper.Info("Successfully close the Redis connection")
	}

	data := &Data{
		db:    db.Model(&Favorite{}), // specify table in advance
		cache: cache,
		log:   logHelper,
	}
	return data, cleanup, nil
}

// NewMysqlConn mysql数据库连接
func NewMysqlConn(c *conf.Data, l log.Logger) *gorm.DB {
	logs := log.NewHelper(log.With(l, "module", "data/mysql"))
	db, err := gorm.Open(mysql.Open(c.Mysql.Dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		logs.Fatalf("Database connection failure, err : %v", err)
	}
	InitDB(db)
	logs.Info("Database enabled successfully!")
	return db.Model(&Favorite{})
}

// NewRedisConn Redis数据库连接, 并发开启连接提高速率
func NewRedisConn(c *conf.Data, l log.Logger) (cacheClient *redis.Client) {
	logs := log.NewHelper(log.With(l, "module", "data/redis"))
	// 初始化点赞数Redis客户端
	cache := redis.NewClient(&redis.Options{
		DB:           int(c.Redis.FavoriteDb),
		Addr:         c.Redis.Addr,
		Username:     c.Redis.Username,
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		Password:     c.Redis.Password,
	})
	// ping Redis客户端，判断连接是否存在
	_, err := cache.Ping(context.Background()).Result()
	if err != nil {
		logs.Fatalf("Redis database connection failure, err : %v", err)
	}
	logs.Info("Cache enabled successfully!")
	return cache
}

// InitDB 创建User数据表，并自动迁移
func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Favorite{}); err != nil {
		log.Fatalf("Database initialization error, err : %v", err)
	}
}
