package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/toomanysource/atreus/app/favorite/service/internal/conf"
	"github.com/toomanysource/atreus/app/favorite/service/internal/server"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

var ctx = context.Background()
var (
	fRepo            *favoriteRepo
	testFavoriteData = []Favorite{
		// user 1
		{
			VideoID: 1,
			UserID:  1,
		},
		{
			VideoID: 2,
			UserID:  1,
		},
		{
			VideoID: 3,
			UserID:  1,
		},
		// other user
		{
			VideoID: 1,
			UserID:  2,
		},
		{
			VideoID: 2,
			UserID:  2,
		},
		{
			VideoID: 1,
			UserID:  3,
		},
	}
)

func TestMain(m *testing.M) {
	db := NewMysqlConn(testConfig, log.DefaultLogger)
	cache := NewRedisConn(testConfig, log.DefaultLogger)
	logger := log.DefaultLogger
	kfk := NewKafkaWriter(testConfig)
	publishConn := server.NewPublishClient(testClientConfig, logger)
	data, f, err := NewData(db, cache, kfk, logger)
	if err != nil {
		panic(err)
	}
	fRepo = (NewFavoriteRepo(data, publishConn, logger)).(*favoriteRepo)
	r := m.Run()
	time.Sleep(time.Second * 2)
	f()
	os.Exit(r)
}

var testConfig = &conf.Data{
	Mysql: &conf.Data_Mysql{
		Driver: "mysql",
		// if you don't use default config, the source must be modified
		Dsn: "root:toomanysource@tcp(127.0.0.1:3306)/atreus?charset=utf8mb4&parseTime=True&loc=Local",
	},
	Redis: &conf.Data_Redis{
		FavoriteDb:   8,
		Addr:         "127.0.0.1:6379",
		Password:     "atreus",
		ReadTimeout:  &durationpb.Duration{Seconds: 1},
		WriteTimeout: &durationpb.Duration{Seconds: 1},
	},
}

var testClientConfig = &conf.Client{
	User: &conf.Client_User{
		To: "0.0.0.0:9005",
	},
	Publish: &conf.Client_Publish{
		To: "0.0.0.0:9003",
	},
}

func Test_favoriteRepo_CreateFavorite(t *testing.T) {
	for _, v := range testFavoriteData {
		err := fRepo.CreateFavorite(ctx, v.UserID, v.VideoID)
		if err != nil {
			t.Error(err)
		}
	}
}

func Test_favoriteRepo_GetFavoriteList(t *testing.T) {
	result, err := fRepo.GetFavoriteList(ctx, 1)
	assert.Nil(t, err)
	assert.Equal(t, len(result), 3)
}

func Test_favoriteRepo_DeleteFavorite(t *testing.T) {
	for i := 0; i < 3; i++ {
		v := testFavoriteData[i]
		err := fRepo.DeleteFavorite(ctx, v.UserID, v.VideoID)
		assert.Nil(t, err)
	}
}

func Test_favoriteRepo_IsFavorite(t *testing.T) {
	isFavorite, err := fRepo.IsFavorite(ctx, 3, []uint32{1})
	assert.Nil(t, err)
	assert.Equal(t, isFavorite, []bool{true})
}
