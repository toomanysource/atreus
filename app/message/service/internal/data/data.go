package data

import (
	"context"

	"github.com/toomanysource/atreus/app/message/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"github.com/segmentio/kafka-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ProviderSet = wire.NewSet(NewData, NewMessageRepo, NewMysqlConn, NewKafkaConn, NewRedisConn)

type KafkaConn struct {
	writer *kafka.Writer
	reader *kafka.Reader
}

type Data struct {
	db    *gorm.DB
	cache *redis.Client
	kfk   *KafkaConn
	log   *log.Helper
}

func NewData(db *gorm.DB, kfk *KafkaConn, cache *redis.Client, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "data/comment"))

	cleanup := func() {
		_, err := cache.Ping(context.Background()).Result()
		if err != nil {
			return
		}
		if err = cache.Close(); err != nil {
			logHelper.Errorf("Redis connection closure failed, err: %w", err)
		}
		logHelper.Info("[redis] client stopping")

		if err := kfk.writer.Close(); err != nil {
			logHelper.Errorf("Kafka connection closure failed, err: %w", err)
		}
		if err := kfk.reader.Close(); err != nil {
			logHelper.Errorf("Kafka connection closure failed, err: %w", err)
		}
		logHelper.Info("[kafka] client stopping")
	}

	data := &Data{
		db:    db.Model(&Message{}),
		kfk:   kfk,
		cache: cache,
		log:   logHelper,
	}
	return data, cleanup, nil
}

// NewMysqlConn mysql数据库连接
func NewMysqlConn(c *conf.Data) *gorm.DB {
	db, err := gorm.Open(mysql.Open(c.Mysql.Dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Database connection failure, err : %v", err)
	}
	InitDB(db)
	log.Info("Database enabled successfully!")
	return db
}

func NewKafkaConn(c *conf.Data) *KafkaConn {
	writer := kafka.Writer{
		Addr:                   kafka.TCP(c.Kafka.Addr),
		Topic:                  c.Kafka.Topic,
		Balancer:               &kafka.LeastBytes{},
		WriteTimeout:           c.Kafka.WriteTimeout.AsDuration(),
		ReadTimeout:            c.Kafka.ReadTimeout.AsDuration(),
		AllowAutoTopicCreation: true,
	}
	var maxBytes int = 10e6
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{c.Kafka.Addr},
		Partition: int(c.Kafka.Partition),
		GroupID:   "store",
		Topic:     c.Kafka.Topic,
		MaxBytes:  maxBytes, // 10MB
	})
	log.Info("Kafka enabled successfully!")
	return &KafkaConn{
		writer: &writer,
		reader: reader,
	}
}

// NewRedisConn Redis数据库连接
func NewRedisConn(c *conf.Data, l log.Logger) (cacheClient *redis.Client) {
	logs := log.NewHelper(log.With(l, "module", "data/redis"))
	// 初始化聊天记录Redis客户端
	cache := redis.NewClient(&redis.Options{
		DB:           int(c.Redis.MessageDb),
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

// InitDB 创建followers数据表，并自动迁移
func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Message{}); err != nil {
		log.Fatalf("Database initialization error, err : %v", err)
	}
}
