package data

import (
	"context"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/toomanysource/atreus/app/publish/service/internal/conf"
	"github.com/toomanysource/atreus/pkg/minioX"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewKafkaReader, NewKafkaWriter, NewPublishRepo, NewMysqlConn, NewRedisConn, NewMinioConn, NewMinioExtraConn, NewMinioIntraConn)

type KfkReader struct {
	comment  *kafka.Reader
	favorite *kafka.Reader
}

// Data .
type Data struct {
	db        *gorm.DB
	oss       *minioX.Client
	kfkReader KfkReader
	kfkWriter *kafka.Writer
	cache     *redis.Client
	log       *log.Helper
}

// NewData .
func NewData(db *gorm.DB, minioClient *minioX.Client, kfkWriter *kafka.Writer, kfkReader KfkReader, cacheClient *redis.Client, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfkReader.comment.Close(); err != nil {
				log.NewHelper(logger).Errorf("Kafka connection closure failed, err: %w", err)
			}
		}()
		wg.Add(1)
		go func() {
			if err := kfkReader.favorite.Close(); err != nil {
				log.NewHelper(logger).Errorf("Kafka connection closure failed, err: %w", err)
			}
		}()
		wg.Add(1)
		go func() {
			if err := kfkWriter.Close(); err != nil {
				log.NewHelper(logger).Errorf("Kafka connection closure failed, err: %w", err)
			}
		}()
		wg.Wait()
		log.NewHelper(logger).Info("Successfully close the Kafka connection")
	}
	data := &Data{
		db:        db.Model(&Video{}),
		oss:       minioClient,
		kfkReader: kfkReader,
		kfkWriter: kfkWriter,
		cache:     cacheClient,
		log:       log.NewHelper(logger),
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

// NewRedisConn Redis数据库连接
func NewRedisConn(c *conf.Data) *redis.Client {
	client := redis.NewClient(&redis.Options{
		DB:           int(c.Redis.Db),
		Addr:         c.Redis.Addr,
		Username:     "atreus",
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		Password:     c.Redis.Password,
	})

	// ping Redis客户端，判断连接是否存在
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Redis database connection failure, err : %v", err)
	}
	log.Info("Cache enabled successfully!")
	return client
}

func NewMinioConn(c *conf.Minio, extraConn minioX.ExtraConn, intraConn minioX.IntraConn) *minioX.Client {
	client := minioX.NewClient(extraConn, intraConn)
	if exists, err := client.ExistBucket(context.Background(), c.BucketName); !exists || err != nil {
		log.Fatalf("Minio bucket %s miss,err: %v", c.BucketName, err)
	}
	return client
}

func NewMinioExtraConn(c *conf.Minio) minioX.ExtraConn {
	extraConn, err := minio.New(c.EndpointExtra, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKeyId, c.AccessSecret, ""),
		Secure: c.UseSsl,
	})
	if err != nil {
		log.Fatalf("minio client init failed,err: %v", err)
	}
	log.Info("minioExtra enabled successfully")
	return minioX.NewExtraConn(extraConn)
}

func NewMinioIntraConn(c *conf.Minio) minioX.IntraConn {
	intraConn, err := minio.New(c.EndpointIntra, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKeyId, c.AccessSecret, ""),
		Secure: c.UseSsl,
	})
	if err != nil {
		log.Fatalf("minio client init failed,err: %v", err)
	}
	log.Info("minioIntra enabled successfully")
	return minioX.NewIntraConn(intraConn)
}

func NewKafkaReader(c *conf.Data) KfkReader {
	var maxBytes int = 10e6
	commentReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{c.Kafka.Addr},
		Partition: int(c.Kafka.Partition),
		GroupID:   "comment",
		Topic:     c.Kafka.CommentTopic,
		MaxBytes:  maxBytes, // 10MB
	})
	log.Info("Kafka enabled successfully!")
	favoriteReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{c.Kafka.Addr},
		Partition: int(c.Kafka.Partition),
		GroupID:   "favorite",
		Topic:     c.Kafka.FavoriteTopic,
		MaxBytes:  maxBytes, // 10MB
	})
	log.Info("Kafka enabled successfully!")
	return KfkReader{
		comment:  commentReader,
		favorite: favoriteReader,
	}
}

func NewKafkaWriter(c *conf.Data) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(c.Kafka.Addr),
		Topic:                  c.Kafka.PublishTopic,
		Balancer:               &kafka.LeastBytes{},
		WriteTimeout:           c.Kafka.WriteTimeout.AsDuration(),
		ReadTimeout:            c.Kafka.ReadTimeout.AsDuration(),
		AllowAutoTopicCreation: true,
	}
}

func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Video{}); err != nil {
		log.Fatalf("Database initialization error, err : %v", err)
	}
}
