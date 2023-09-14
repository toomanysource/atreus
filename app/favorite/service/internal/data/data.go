package data

import (
	"context"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/app/favorite/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ProviderSet = wire.NewSet(NewData, NewKafkaWriter, NewFavoriteRepo, NewPublishRepo, NewMysqlConn, NewRedisConn)

type KfkWriter struct {
	Favorite      *kafka.Writer
	Favored       *kafka.Writer
	videoFavorite *kafka.Writer
}

type Data struct {
	db    *gorm.DB
	cache *redis.Client
	kfk   KfkWriter
	log   *log.Helper
}

func NewData(db *gorm.DB, cache *redis.Client, kfk KfkWriter, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "data/data"))
	// 并发关闭所有数据库连接
	cleanup := func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.Ping(context.Background()).Result()
			if err != nil {
				logHelper.Warn("redis connection pool is empty")
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
			if err := kfk.Favored.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the kafka favored queue connection")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.Favorite.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the kafka favorite queue connection")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfk.videoFavorite.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the kafka video favorite queue connection")
		}()
		wg.Wait()
	}

	data := &Data{
		db:    db.Model(&Favorite{}), // specify table in advance
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
	logs.Info("database enabled successfully!")
	return db
}

// NewRedisConn Redis数据库连接, 并发开启连接提高速率
func NewRedisConn(c *conf.Data, l log.Logger) (cacheClient *redis.Client) {
	logs := log.NewHelper(log.With(l, "module", "data/data/redis"))
	// 初始化点赞数Redis客户端
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

func NewKafkaWriter(c *conf.Data, l log.Logger) KfkWriter {
	logs := log.NewHelper(log.With(l, "module", "data/data/kafka"))
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
		Favorite:      writer(c.Kafka.FavoriteTopic),
		Favored:       writer(c.Kafka.FavoredTopic),
		videoFavorite: writer(c.Kafka.VideoFavoriteTopic),
	}
}

// InitDB 创建User数据表，并自动迁移
func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Favorite{}); err != nil {
		log.Fatalf("database initialization error, err : %v", err)
	}
}
