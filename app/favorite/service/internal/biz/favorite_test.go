package biz

import (
	"context"
	"os"
	"testing"

	"github.com/toomanysource/atreus/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

var (
	ctx           = context.Background()
	testVideoData = []Video{
		{
			Id:            1,
			Author:        &User{Id: 1},
			PlayUrl:       "https://www.baidu.com",
			CoverUrl:      "https://www.baidu.com",
			FavoriteCount: 2,
			CommentCount:  1,
			IsFavorite:    true,
			Title:         "test1",
		},
		{
			Id:            2,
			Author:        &User{Id: 1},
			PlayUrl:       "https://www.baidu.com",
			CoverUrl:      "https://www.baidu.com",
			FavoriteCount: 1,
			CommentCount:  1,
			IsFavorite:    false,
			Title:         "test2",
		},
		{
			Id:            3,
			Author:        &User{Id: 1},
			PlayUrl:       "https://www.baidu.com",
			CoverUrl:      "https://www.baidu.com",
			FavoriteCount: 1,
			CommentCount:  1,
			IsFavorite:    false,
			Title:         "test3",
		},
	}
)

type MockFavoriteRepo struct{}

func (m *MockFavoriteRepo) DeleteFavorite(ctx context.Context, userId uint32, videoId uint32) error {
	return nil
}

func (m *MockFavoriteRepo) CreateFavorite(ctx context.Context, userId uint32, videoId uint32) error {
	return nil
}

func (m *MockFavoriteRepo) GetFavoriteList(ctx context.Context, userId uint32) ([]Video, error) {
	var favoriteList []Video
	for _, v := range testVideoData {
		if v.IsFavorite == true {
			favoriteList = append(favoriteList, v)
		}
	}
	return favoriteList, nil
}

func (m *MockFavoriteRepo) IsFavorite(ctx context.Context, userId uint32, videoId []uint32) ([]bool, error) {
	isFavorite := make([]bool, len(videoId))
	for i := range videoId {
		isFavorite[i] = false
	}
	return isFavorite, nil
}

var (
	mockRepo = &MockFavoriteRepo{}
	usecase  *FavoriteUseCase
)

func TestMain(m *testing.M) {
	ctx = context.WithValue(ctx, middleware.UserIdKey("user_id"), uint32(1))
	usecase = NewFavoriteUseCase(mockRepo, log.DefaultLogger)
	r := m.Run()
	os.Exit(r)
}

func TestFavoriteUsecase_FavoriteAction(t *testing.T) {
	err := usecase.FavoriteAction(
		ctx, 1, 2)
	assert.Nil(t, err)
	err = usecase.FavoriteAction(
		ctx, 1, 1)
	assert.Nil(t, err)
	err = usecase.FavoriteAction(
		ctx, 1, 3)
	assert.NotEqual(t, err, nil)
}

func TestFavoriteUsecase_GetFavoriteList(t *testing.T) {
	favorites, err := usecase.GetFavoriteList(context.TODO(), 1)
	assert.Nil(t, err)
	for _, v := range favorites {
		assert.Equal(t, v.IsFavorite, true)
	}
}

func TestFavoriteUsecase_IsFavorite(t *testing.T) {
	isFavorite, err := usecase.IsFavorite(context.TODO(), 1, []uint32{6})
	assert.Nil(t, err)
	assert.Equal(t, isFavorite[0], false)
	isFavorite, err = usecase.IsFavorite(context.TODO(), 1, []uint32{1})
	assert.Nil(t, err)
	assert.Equal(t, isFavorite[0], false)
}
