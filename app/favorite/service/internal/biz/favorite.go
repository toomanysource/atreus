package biz

import (
	"context"
	"fmt"

	"github.com/toomanysource/atreus/middleware"

	"github.com/go-kratos/kratos/v2/log"
)

var (
	actionFavorite   uint32 = 1
	actionUnFavorite uint32 = 2
)

type Video struct {
	Id            uint32
	Author        *User
	PlayUrl       string
	CoverUrl      string
	FavoriteCount uint32
	CommentCount  uint32
	IsFavorite    bool
	Title         string
}

type User struct {
	Id              uint32
	Name            string
	FollowCount     uint32
	FollowerCount   uint32
	IsFollow        bool
	Avatar          string
	BackgroundImage string
	Signature       string
	TotalFavorited  uint32 // 总获赞数
	WorkCount       uint32
	FavoriteCount   uint32 // 点赞数量
}

type FavoriteRepo interface {
	GetFavoriteList(ctx context.Context, userID uint32) ([]Video, error)
	IsFavorite(ctx context.Context, userID uint32, videoID []uint32) ([]bool, error)
	DeleteFavorite(ctx context.Context, userID uint32, videoID uint32) error
	CreateFavorite(ctx context.Context, userID uint32, videoID uint32) error
}

type PublishRepo interface {
	GetVideoListByVideoIds(ctx context.Context, userId uint32, videoIds []uint32) ([]Video, error)
}

type FavoriteUseCase struct {
	favoriteRepo FavoriteRepo
	log          *log.Helper
}

func NewFavoriteUseCase(repo FavoriteRepo, logger log.Logger) *FavoriteUseCase {
	return &FavoriteUseCase{favoriteRepo: repo, log: log.NewHelper(log.With(logger, "model", "usecase/favorite"))}
}

func (uc *FavoriteUseCase) FavoriteAction(ctx context.Context, videoId, actionType uint32) error {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	switch actionType {
	case actionFavorite:
		return uc.favoriteRepo.CreateFavorite(ctx, userId, videoId)
	case actionUnFavorite:
		return uc.favoriteRepo.DeleteFavorite(ctx, userId, videoId)
	default:
		return fmt.Errorf("invalid action type(not 1 nor 2)")
	}
}

func (uc *FavoriteUseCase) GetFavoriteList(ctx context.Context, userID uint32) ([]Video, error) {
	userIdFromToken := ctx.Value("user_id").(uint32)
	if userIdFromToken != userID {
		uc.log.Errorf(
			"GetFavoriteList: userID not correspond to token,token: %d, param:%d", userIdFromToken, userID)
		return nil, fmt.Errorf("invalid token")
	}
	return uc.favoriteRepo.GetFavoriteList(ctx, userID)
}

func (uc *FavoriteUseCase) IsFavorite(ctx context.Context, userID uint32, videoIDs []uint32) ([]bool, error) {
	ret, err := uc.favoriteRepo.IsFavorite(ctx, userID, videoIDs)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
