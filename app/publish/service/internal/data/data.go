package data

import (
	"context"
	"errors"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/toomanysource/atreus/app/publish/service/internal/conf"
	"github.com/toomanysource/atreus/pkg/minioX"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ProviderSet = wire.NewSet(NewData, NewKafkaReader, NewKafkaWriter, NewPublishRepo, NewMysqlConn, NewMinioConn, NewMinioExtraConn, NewMinioIntraConn)

var (
	ErrCopy                    = errors.New("copy error")
	ErrMysqlInsert             = errors.New("mysql insert error")
	ErrMysqlQuery              = errors.New("mysql query error")
	ErrUserServiceResponse     = errors.New("user service response error")
	ErrKafkaReader             = errors.New("kafka reader error")
	ErrFileCreate              = errors.New("file create error")
	ErrFileRead                = errors.New("file read error")
	ErrFileWrite               = errors.New("file write error")
	ErrMysqlUpdate             = errors.New("mysql update error")
	ErrFavoriteServiceResponse = errors.New("favorite service response error")
)

type KfkReader struct {
	comment  *kafka.Reader
	favorite *kafka.Reader
}

type Data struct {
	db        *gorm.DB
	oss       *minioX.Client
	kfkReader KfkReader
	kfkWriter *kafka.Writer
	log       *log.Helper
}

func NewData(db *gorm.DB, minioClient *minioX.Client, kfkWriter *kafka.Writer, kfkReader KfkReader, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "data/data"))
	cleanup := func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfkReader.comment.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the kafka comment queue connection")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfkReader.favorite.Close(); err != nil {
				logHelper.Errorf("kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the kafka favorite queue connection")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := kfkWriter.Close(); err != nil {
				logHelper.Errorf("Kafka connection closure failed, err: %w", err)
			}
			logHelper.Info("successfully close the kafka writer connection")
		}()
		wg.Wait()
	}
	data := &Data{
		db:        db.Model(&Video{}),
		oss:       minioClient,
		kfkReader: kfkReader,
		kfkWriter: kfkWriter,
		log:       logHelper,
	}
	return data, cleanup, nil
}

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

func NewMinioConn(c *conf.Minio, extraConn minioX.ExtraConn, intraConn minioX.IntraConn, l log.Logger) *minioX.Client {
	logs := log.NewHelper(log.With(l, "module", "data/data/minio"))
	client := minioX.NewClient(extraConn, intraConn)
	if err := client.CreateBucket(context.Background(), c.BucketName); err != nil {
		logs.Fatal(err)
	}
	logs.Info("minio enabled successfully")
	return client
}

func NewMinioExtraConn(c *conf.Minio, l log.Logger) minioX.ExtraConn {
	logs := log.NewHelper(log.With(l, "module", "data/data/minioExtra"))
	extraConn, err := minio.New(c.EndpointExtra, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKeyId, c.AccessSecret, ""),
		Secure: c.UseSsl,
	})
	if err != nil {
		logs.Fatalf("minio client init failed,err: %v", err)
	}
	return minioX.NewExtraConn(extraConn)
}

func NewMinioIntraConn(c *conf.Minio, l log.Logger) minioX.IntraConn {
	logs := log.NewHelper(log.With(l, "module", "data/data/minioIntra"))
	intraConn, err := minio.New(c.EndpointIntra, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKeyId, c.AccessSecret, ""),
		Secure: c.UseSsl,
	})
	if err != nil {
		logs.Fatalf("minio client init failed,err: %v", err)
	}
	return minioX.NewIntraConn(intraConn)
}

func NewKafkaReader(c *conf.Data, l log.Logger) KfkReader {
	logs := log.NewHelper(log.With(l, "module", "data/data/kafkaReader"))
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
	logs.Info("kafka reader enabled successfully")
	return KfkReader{
		comment:  reader(c.Kafka.CommentTopic),
		favorite: reader(c.Kafka.FavoriteTopic),
	}
}

func NewKafkaWriter(c *conf.Data, l log.Logger) *kafka.Writer {
	logs := log.NewHelper(log.With(l, "module", "data/data/kafkaWriter"))
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(c.Kafka.Addr),
		Topic:                  c.Kafka.PublishTopic,
		Balancer:               &kafka.LeastBytes{},
		WriteTimeout:           c.Kafka.WriteTimeout.AsDuration(),
		ReadTimeout:            c.Kafka.ReadTimeout.AsDuration(),
		AllowAutoTopicCreation: true,
	}
	logs.Info("kafka writer enabled successfully")
	return writer
}

func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Video{}); err != nil {
		log.Fatalf("database initialization error, err : %v", err)
	}
}
