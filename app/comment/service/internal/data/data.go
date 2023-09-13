package data

import (
	"context"
	"sync"

	"github.com/go-redis/redis/v8"

	"github.com/segmentio/kafka-go"

	"github.com/toomanysource/atreus/app/comment/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ProviderSet = wire.NewSet(
	NewData, NewKafkaWriter, NewCommentRepo, NewMysqlConn, NewRedisConn, NewUserRepo,
)

type Data struct {
	kfk *kafka.Writer
	log *log.Helper
}

func NewData(cacheClient *redis.Client, kfk *kafka.Writer, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "data/data"))
	// 并发关闭所有数据库连接
	cleanup := func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cacheClient.Ping(context.Background()).Result()
			if err != nil {
				logHelper.Warn("redis connection pool is empty")
				return
			}
			if err = cacheClient.Close(); err != nil {
				logHelper.Errorf("redis connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the redis connection")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the kafka connection")
		}()
		wg.Wait()
	}

	data := &Data{
		kfk: kfk,
		log: logHelper,
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
	return db.Model(&Comment{})
}

// NewRedisConn Redis数据库连接
func NewRedisConn(c *conf.Data, l log.Logger) *redis.Client {
	logs := log.NewHelper(log.With(l, "module", "data/data/redis"))
	client := redis.NewClient(&redis.Options{
		DB:           int(c.Redis.Db),
		Addr:         c.Redis.Addr,
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		Password:     c.Redis.Password,
	})

	// ping Redis客户端，判断连接是否存在
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		logs.Fatalf("cache connection failure, err : %v", err)
	}
	logs.Info("cache enabled successfully")
	return client
}

func NewKafkaWriter(c *conf.Data, l log.Logger) *kafka.Writer {
	logs := log.NewHelper(log.With(l, "module", "data/data/kafka"))
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(c.Kafka.Addr),
		Topic:                  c.Kafka.Topic,
		Balancer:               &kafka.LeastBytes{},
		WriteTimeout:           c.Kafka.WriteTimeout.AsDuration(),
		ReadTimeout:            c.Kafka.ReadTimeout.AsDuration(),
		AllowAutoTopicCreation: true,
	}
	logs.Info("kafka enabled successfully")
	return writer
}

// InitDB 创建Comments数据表，并自动迁移
func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Comment{}); err != nil {
		log.Fatalf("database initialization error, err : %v", err)
	}
}
