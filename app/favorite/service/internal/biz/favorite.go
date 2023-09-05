package biz

import (
	"context"

	"github.com/toomanysource/atreus/middleware"
	"github.com/toomanysource/atreus/pkg/errorX"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	Favorite   uint32 = 1
	UnFavorite uint32 = 2
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
	repo FavoriteRepo
	log  *log.Helper
}

func NewFavoriteUseCase(repo FavoriteRepo, logger log.Logger) *FavoriteUseCase {
	return &FavoriteUseCase{repo: repo, log: log.NewHelper(log.With(logger, "model", "usecase/favorite"))}
}

func (uc *FavoriteUseCase) FavoriteAction(ctx context.Context, videoId, actionType uint32) error {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	switch actionType {
	case Favorite:
		return uc.repo.CreateFavorite(ctx, userId, videoId)
	case UnFavorite:
		return uc.repo.DeleteFavorite(ctx, userId, videoId)
	default:
		return errorX.ErrInValidActionType
	}
}

func (uc *FavoriteUseCase) GetFavoriteList(ctx context.Context, userID uint32) ([]Video, error) {
	if userID == 0 {
		userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
		return uc.repo.GetFavoriteList(ctx, userId)
	}
	return uc.repo.GetFavoriteList(ctx, userID)
}

func (uc *FavoriteUseCase) IsFavorite(ctx context.Context, userID uint32, videoIDs []uint32) ([]bool, error) {
	return uc.repo.IsFavorite(ctx, userID, videoIDs)
}
