package data

import (
	"context"
	"errors"
	"sync"

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

var (
	ErrCopy             = errors.New("copy error")
	ErrJsonMarshal      = errors.New("json marshal error")
	ErrRedisSet         = errors.New("redis set error")
	ErrRedisQuery       = errors.New("redis query error")
	ErrMysqlInsert      = errors.New("mysql insert error")
	ErrMysqlQuery       = errors.New("mysql query error")
	ErrRedisDelete      = errors.New("redis delete error")
	ErrRedisTransaction = errors.New("redis transaction error")
)

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
	logHelper := log.NewHelper(log.With(logger, "module", "data/data"))

	cleanup := func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.Ping(context.Background()).Result()
			if err != nil {
				return
			}
			if err = cache.Close(); err != nil {
				logHelper.Errorf("redis connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the redis connection")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.writer.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the kafka writer connection")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.reader.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the kafka reader connection")
		}()
		wg.Wait()
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

func NewKafkaConn(c *conf.Data, l log.Logger) *KafkaConn {
	logs := log.NewHelper(log.With(l, "module", "data/data/kafka"))
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
	logs.Info("kafka enabled successfully")
	return &KafkaConn{
		writer: &writer,
		reader: reader,
	}
}

// NewRedisConn Redis数据库连接
func NewRedisConn(c *conf.Data, l log.Logger) (cacheClient *redis.Client) {
	logs := log.NewHelper(log.With(l, "module", "data/data/redis"))
	// 初始化聊天记录Redis客户端
	cache := redis.NewClient(&redis.Options{
		DB:           int(c.Redis.Db),
		Addr:         c.Redis.Addr,
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		Password:     c.Redis.Password,
	})
	// ping Redis客户端，判断连接是否存在
	_, err := cache.Ping(context.Background()).Result()
	if err != nil {
		logs.Fatalf("redis database connection failure, err : %v", err)
	}
	logs.Info("cache enabled successfully")
	return cache
}

// InitDB 创建followers数据表，并自动迁移
func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Message{}); err != nil {
		log.Fatalf("database initialization error, err : %v", err)
	}
}
