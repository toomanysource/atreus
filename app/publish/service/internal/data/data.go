package data

import (
	"context"

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
var ProviderSet = wire.NewSet(NewData, NewPublishRepo, NewMysqlConn, NewRedisConn, NewMinioConn, NewMinioExtraConn, NewMinioIntraConn)

// Data .
type Data struct {
	db    *gorm.DB
	oss   *minioX.Client
	cache *redis.Client
	log   *log.Helper
}

// NewData .
func NewData(db *gorm.DB, minioClient *minioX.Client, cacheClient *redis.Client, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	data := &Data{
		db:    db.Model(&Video{}),
		oss:   minioClient,
		cache: cacheClient,
		log:   log.NewHelper(logger),
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

func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Video{}); err != nil {
		log.Fatalf("Database initialization error, err : %v", err)
	}
}
