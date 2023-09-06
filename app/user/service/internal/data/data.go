package data

import (
	"context"
	"sync"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm/logger"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/app/user/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewKafkaReader, NewGormDb, NewRedisConn, NewUserRepo, NewRelationRepo)

var (
	maxOpenConnection = 100
	maxIdleConnecton  = 10
)

type KfkReader struct {
	follow   *kafka.Reader
	follower *kafka.Reader
	favorite *kafka.Reader
	favored  *kafka.Reader
	publish  *kafka.Reader
}

// Data .
type Data struct {
	db  *gorm.DB
	kfk KfkReader
	rdb *redis.Client
	log *log.Helper
}

// NewData .
func NewData(db *gorm.DB, rdb *redis.Client, kfk KfkReader, logger log.Logger) (*Data, func(), error) {
	logs := log.NewHelper(log.With(logger, "resources", "data"))
	cleanup := func() {
		var wg sync.WaitGroup
		logs.Info("closing the data resources")
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := rdb.Close(); err != nil {
				logs.Errorf("close redis connection failed: %w", err)
				return
			}
			logs.Infof("close redis connection successfully")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.publish.Close(); err != nil {
				logs.Errorf("close kafka connection failed: %w", err)
				return
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.follow.Close(); err != nil {
				logs.Errorf("close kafka connection failed: %w", err)
				return
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.follower.Close(); err != nil {
				logs.Errorf("close kafka connection failed: %w", err)
				return
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.favorite.Close(); err != nil {
				logs.Errorf("close kafka connection failed: %w", err)
				return
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.favored.Close(); err != nil {
				logs.Errorf("close kafka connection failed: %w", err)
				return
			}
		}()
		wg.Wait()
		logs.Info("Successfully close the data resources")
	}
	data := &Data{
		db:  db,
		rdb: rdb,
		kfk: kfk,
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

func NewKafkaReader(c *conf.Data) KfkReader {
	var maxBytes int = 10e6
	reader := func(topic string) *kafka.Reader {
		return kafka.NewReader(kafka.ReaderConfig{
			Brokers:   []string{c.Kafka.Addr},
			Topic:     topic,
			Partition: int(c.Kafka.Partition),
			GroupID:   topic,
			MaxBytes:  maxBytes, // 10MB
		})
	}
	return KfkReader{
		follow:   reader(c.Kafka.FollowTopic),
		follower: reader(c.Kafka.FollowerTopic),
		favorite: reader(c.Kafka.FavoriteTopic),
		favored:  reader(c.Kafka.FavoredTopic),
		publish:  reader(c.Kafka.PublishTopic),
	}
}
