package biz

import (
	"context"

	"github.com/toomanysource/atreus/middleware"

	"github.com/go-kratos/kratos/v2/log"
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
		err := uc.repo.CreateFavorite(ctx, userId, videoId)
		if err != nil {
			uc.log.Errorf("create favorite error: %v", err)
		}
		return err
	case UnFavorite:
		err := uc.repo.DeleteFavorite(ctx, userId, videoId)
		if err != nil {
			uc.log.Errorf("delete favorite error: %v", err)
		}
		return err
	default:
		return ErrInValidActionType
	}
}

func (uc *FavoriteUseCase) GetFavoriteList(ctx context.Context, userID uint32) ([]Video, error) {
	if userID == 0 {
		userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
		videos, err := uc.repo.GetFavoriteList(ctx, userId)
		if err != nil {
			uc.log.Errorf("GetFavoriteList error: %v", err)
		}
		return videos, err
	}
	videos, err := uc.repo.GetFavoriteList(ctx, userID)
	if err != nil {
		uc.log.Errorf("GetFavoriteList error: %v", err)
	}
	return videos, err
}

func (uc *FavoriteUseCase) IsFavorite(ctx context.Context, userID uint32, videoIDs []uint32) ([]bool, error) {
	oks, err := uc.repo.IsFavorite(ctx, userID, videoIDs)
	if err != nil {
		uc.log.Errorf("IsFavorite error: %v", err)
	}
	return oks, err
}
