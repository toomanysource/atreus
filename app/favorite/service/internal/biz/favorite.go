package biz

import (
	"context"
	"errors"

	"github.com/toomanysource/atreus/app/favorite/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

// Video is used to receive video info from video service;response is an array of Videos
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

// User is used to receive video info from user service;& send response
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

type UserRepo interface {
	UpdateFavorited(ctx context.Context, userId uint32, change int32) error
	UpdateFavorite(ctx context.Context, userId uint32, change int32) error
}

type PublishRepo interface {
	GetVideoListByVideoIds(ctx context.Context, userId uint32, videoIds []uint32) ([]Video, error)
	UpdateFavoriteCount(ctx context.Context, videoId uint32, change int32) error
}

type FavoriteUsecase struct {
	favoriteRepo FavoriteRepo
	config       *conf.JWT
	log          *log.Helper
}

func NewFavoriteUsecase(conf *conf.JWT, repo FavoriteRepo, logger log.Logger) *FavoriteUsecase {
	return &FavoriteUsecase{config: conf, favoriteRepo: repo, log: log.NewHelper(log.With(logger, "model", "usecase/favorite"))}
}

func (uc *FavoriteUsecase) FavoriteAction(ctx context.Context, videoId, actionType uint32, tokenString string) error {
	userId := ctx.Value("user_id").(uint32)
	switch actionType {
	case 1:
		return uc.favoriteRepo.CreateFavorite(ctx, userId, videoId)
	case 2:
		return uc.favoriteRepo.DeleteFavorite(ctx, userId, videoId)
	default:
		return errors.New("invalid action type(not 1 nor 2)")
	}
}

func (uc *FavoriteUsecase) GetFavoriteList(ctx context.Context, userID uint32, tokenString string) ([]Video, error) {
	userIdFromToken := ctx.Value("user_id").(uint32)
	if userIdFromToken != userID {
		uc.log.Errorf(
			"GetFavoriteList: userID not correspond to token,token: %d, param:%d", userIdFromToken, userID)
	}
	return uc.favoriteRepo.GetFavoriteList(ctx, userID)
}

func (uc *FavoriteUsecase) IsFavorite(ctx context.Context, userID uint32, videoIDs []uint32) ([]bool, error) {
	ret, err := uc.favoriteRepo.IsFavorite(ctx, userID, videoIDs)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
