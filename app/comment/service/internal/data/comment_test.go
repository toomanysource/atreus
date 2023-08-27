package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/toomanysource/atreus/app/comment/service/internal/conf"
	"github.com/toomanysource/atreus/app/comment/service/internal/server"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	ctx              = context.Background()
	testCommentsData = []*Comment{
		{
			Id:       1,
			UserId:   1,
			Content:  "bushuwu1",
			CreateAt: "08-01",
		},
		{
			Id:       2,
			UserId:   1,
			Content:  "dadawd",
			CreateAt: "08-02",
		},
		{
			Id:       3,
			UserId:   2,
			Content:  "bdzxvzad",
			CreateAt: "08-03",
		},
		{
			Id:       4,
			UserId:   1,
			Content:  "bvrbr",
			CreateAt: "08-03",
		},
		{
			Id:       5,
			UserId:   3,
			Content:  "bdadawfvrd",
			CreateAt: "08-04",
		},
		{
			Id:       6,
			UserId:   5,
			Content:  "bdafagaagaga",
			CreateAt: "08-05",
		},
	}
)

var testConfig = &conf.Data{
	Mysql: &conf.Data_Mysql{
		Driver: "mysql",
		// if you don't use default config, the source must be modified
		Dsn: "root:toomanysource@tcp(127.0.0.1:3306)/atreus?charset=utf8mb4&parseTime=True&loc=Local",
	},
	Redis: &conf.Data_Redis{
		CommentDb:    1,
		Addr:         "127.0.0.1:6379",
		Password:     "atreus",
		ReadTimeout:  &durationpb.Duration{Seconds: 1},
		WriteTimeout: &durationpb.Duration{Seconds: 1},
	},
}

var testClientConfig = &conf.Client{
	User: &conf.Client_User{
		To: "0.0.0.0:9001",
	},
}

var cRepo *commentRepo

func TestMain(m *testing.M) {
	ctx = context.WithValue(ctx, "user_id", uint32(1))
	logger := log.DefaultLogger
	db := NewMysqlConn(testConfig, logger)
	cache := NewRedisConn(testConfig, logger)
	userConn := server.NewUserClient(testClientConfig, logger)
	publishConn := server.NewPublishClient(testClientConfig, logger)
	data, f, err := NewData(db, cache, logger)
	if err != nil {
		panic(err)
	}
	cRepo = (NewCommentRepo(data, userConn, publishConn, logger)).(*commentRepo)
	r := m.Run()
	time.Sleep(time.Second * 2)
	f()
	os.Exit(r)
}

func TestCommentRepo_SearchCommentList(t *testing.T) {
	comments, err := cRepo.SearchCommentList(ctx, 1)
	assert.Nil(t, err)
	assert.Equal(t, len(comments), len(testCommentsData)-1)
}

func TestCommentRepo_InsertComment(t *testing.T) {
	_, err := cRepo.InsertComment(ctx, 2, "wuhu", 2)
	assert.Nil(t, err)
}

func TestCommentRepo_DelComment(t *testing.T) {
	err := cRepo.DelComment(ctx, 2, 19, 2)
	assert.Nil(t, err)
}

func TestCommentRepo_CacheCreateCommentTransaction(t *testing.T) {
	err := cRepo.CacheCreateCommentTransaction(ctx, testCommentsData[:5], 1)
	assert.Nil(t, err)
}

func TestCommentRepo_DeleteComment(t *testing.T) {
	_, err := cRepo.DeleteComment(ctx, 2, 6)
	assert.Nil(t, err)
}

func TestCommentRepo_CreateComment(t *testing.T) {
	_, err := cRepo.CreateComment(ctx, 2, "hahaha")
	assert.Nil(t, err)
}

func TestCommentRepo_GetCommentList(t *testing.T) {
	comments, err := cRepo.GetCommentList(ctx, 1)
	assert.Nil(t, err)
	assert.Equal(t, len(comments), 5)
}
