package biz

import (
	"context"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"

	"github.com/toomanysource/atreus/middleware"
)

type MockPublishRepo struct{}

func (m *MockPublishRepo) FindVideoListByUserId(ctx context.Context, userId uint32) (v []*Video, err error) {
	if userId == 1 {
		v = append(v, &Video{
			ID: 1,
		})
	}
	return
}

func (m *MockPublishRepo) UploadAll(ctx context.Context, video []byte, title string) error {
	return nil
}

func (m *MockPublishRepo) GetFeedList(ctx context.Context, latestTime string) (time int64, v []*Video, err error) {
	if latestTime == "0" {
		v = append(v, &Video{
			ID: 1,
		})
	}
	return
}

func (m *MockPublishRepo) FindVideoListByVideoIds(ctx context.Context, userId uint32, videoIds []uint32) ([]*Video, error) {
	return []*Video{{
		ID: 1,
	}}, nil
}

func (m *MockPublishRepo) InitUpdateFavoriteQueue() {}

func (m *MockPublishRepo) InitUpdateCommentQueue() {}

var (
	ctx      = context.Background()
	mockRepo = &MockPublishRepo{}
	useCase  *PublishUsecase
)

func TestMain(m *testing.M) {
	ctx = context.WithValue(ctx, middleware.UserIdKey("user_id"), uint32(1))
	useCase = NewPublishUsecase(mockRepo, log.DefaultLogger)
	m.Run()
	os.Exit(0)
}

func TestPublishUsecase_GetPublishList(t *testing.T) {
	videos, err := useCase.GetPublishList(ctx, 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(videos))
	videos, err = useCase.GetPublishList(ctx, 0)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(videos))
}

func TestPublishUsecase_GetVideoListByVideoIds(t *testing.T) {
	videos, err := useCase.GetVideoListByVideoIds(ctx, 1, []uint32{1})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(videos))
}

func TestPublishUsecase_FeedList(t *testing.T) {
	time, videos, err := useCase.FeedList(ctx, "0")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(videos))
	assert.Equal(t, int64(0), time)
}

func TestPublishUsecase_PublishAction(t *testing.T) {
	err := useCase.PublishAction(ctx, nil, "haha")
	assert.Nil(t, err)
}
