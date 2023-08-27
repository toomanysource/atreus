package data

import (
	"github.com/toomanysource/atreus/app/user/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGormDb, NewUserRepo)

var (
	maxOpenConnection = 100
	maxIdleConnecton  = 10
)

// Data .
type Data struct {
	db  *gorm.DB
	log *log.Helper
}

// NewData .
func NewData(db *gorm.DB, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	data := &Data{
		db:  db.Model(&User{}),
		log: log.NewHelper(logger),
	}
	return data, cleanup, nil
}

// NewGormDb .
func NewGormDb(c *conf.Data) *gorm.DB {
	dsn := c.Database.Source
	open, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("database connect failed, error: " + err.Error())
	}
	db, _ := open.DB()
	// 连接池配置
	db.SetMaxOpenConns(maxOpenConnection)
	db.SetMaxIdleConns(maxIdleConnecton)
	InitDB(open)
	return open
}

func InitDB(conn *gorm.DB) {
	if err := conn.AutoMigrate(&User{}); err != nil {
		log.Fatalf("Database %s initialization error, err : %s", userTableName, err.Error())
	}
}
