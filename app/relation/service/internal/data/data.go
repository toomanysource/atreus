package data

import (
	"context"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/toomanysource/atreus/app/relation/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ProviderSet = wire.NewSet(NewData, NewKafkaWriter, NewRelationRepo, NewUserRepo, NewMysqlConn, NewRedisConn)

type KfkWriter struct {
	follow   *kafka.Writer
	follower *kafka.Writer
}

// CacheClient relation 服务的 Redis 缓存客户端
type CacheClient struct {
	followRelation   *redis.Client // 用户关注关系缓存
	followedRelation *redis.Client // 用户被关注关系缓存
}

type Data struct {
	db    *gorm.DB
	cache *CacheClient
	kfk   KfkWriter
	log   *log.Helper
}

func NewData(db *gorm.DB, cache *CacheClient, kfk KfkWriter, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "data/data"))
	// 关闭Redis连接
	cleanup := func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.followedRelation.Ping(context.Background()).Result()
			if err != nil {
				return
			}
			if err = cache.followedRelation.Close(); err != nil {
				logHelper.Errorf("redis connection closure failed, err: %w", err)
			}
			logHelper.Info("redis followed connection closure successfully")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.followRelation.Ping(context.Background()).Result()
			if err != nil {
				return
			}
			if err = cache.followRelation.Close(); err != nil {
				logHelper.Errorf("redis connection closure failed, err: %w", err)
			}
			logHelper.Info("redis follow connection closure successfully")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.follow.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("kafka follow queue connection closure successfully")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.follower.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("kafka follower queue connection closure successfully")
		}()
		wg.Wait()
	}

	data := &Data{
		db:    db.Model(&Followers{}),
		cache: cache,
		kfk:   kfk,
		log:   logHelper,
	}
	return data, cleanup, nil
}

// NewMysqlConn mysql数据库连接
func NewMysqlConn(c *conf.Data, l log.Logger) *gorm.DB {
	logs := log.NewHelper(log.With(l, "module", "data/data/mysql"))
	db, err := gorm.Open(mysql.Open(c.Mysql.Dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		logs.Fatalf("database connection failure, err : %v", err)
	}
	InitDB(db)
	logs.Info("database enabled successfully")
	return db
}

// NewRedisConn Redis数据库连接
func NewRedisConn(c *conf.Data, l log.Logger) (cache *CacheClient) {
	logs := log.NewHelper(log.With(l, "module", "data/data/redis"))
	var wg sync.WaitGroup
	cache = &CacheClient{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		cache.followedRelation = redis.NewClient(&redis.Options{
			DB:           int(c.Redis.FollowedDb),
			Addr:         c.Redis.Addr,
			WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
			ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
			Password:     c.Redis.Password,
		})

		// ping Redis客户端，判断连接是否存在
		_, err := cache.followedRelation.Ping(context.Background()).Result()
		if err != nil {
			logs.Fatalf("redis database connection failure, err : %v", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		cache.followRelation = redis.NewClient(&redis.Options{
			DB:           int(c.Redis.FollowDb),
			Addr:         c.Redis.Addr,
			WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
			ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
			Password:     c.Redis.Password,
		})

		// ping Redis客户端，判断连接是否存在
		_, err := cache.followRelation.Ping(context.Background()).Result()
		if err != nil {
			logs.Fatalf("redis database connection failure, err : %v", err)
		}
	}()
	wg.Wait()
	logs.Info("cache enabled successfully")
	return
}

func NewKafkaWriter(c *conf.Data, l log.Logger) KfkWriter {
	logs := log.NewHelper(log.With(l, "module", "data/data/kafkaWriter"))
	writer := func(topic string) *kafka.Writer {
		return &kafka.Writer{
			Addr:                   kafka.TCP(c.Kafka.Addr),
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			WriteTimeout:           c.Kafka.WriteTimeout.AsDuration(),
			ReadTimeout:            c.Kafka.ReadTimeout.AsDuration(),
			AllowAutoTopicCreation: true,
		}
	}
	logs.Info("kafka enabled successfully")
	return KfkWriter{
		follow:   writer(c.Kafka.FollowTopic),
		follower: writer(c.Kafka.FollowerTopic),
	}
}

// InitDB 创建followers数据表，并自动迁移
func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Followers{}); err != nil {
		log.Fatalf("database initialization error, err : %v", err)
	}
}
