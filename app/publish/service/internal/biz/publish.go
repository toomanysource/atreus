package biz

import (
	"context"

	"github.com/toomanysource/atreus/middleware"

	"github.com/go-kratos/kratos/v2/log"
)

type Video struct {
	ID            uint32 `copier:"Id"`
	Author        *User
	PlayUrl       string
	CoverUrl      string
	FavoriteCount uint32
	CommentCount  uint32
	IsFavorite    bool
	Title         string
}

type User struct {
	ID              uint32 `copier:"Id"`
	Name            string
	FollowCount     uint32
	FollowerCount   uint32
	IsFollow        bool
	Avatar          string
	BackgroundImage string
	Signature       string
	TotalFavorited  uint32
	WorkCount       uint32
	FavoriteCount   uint32
}

type PublishRepo interface {
	GetVideosByUserId(context.Context, uint32) ([]*Video, error)
	UploadAll(context.Context, []byte, string) error
	GetFeedList(context.Context, string) (int64, []*Video, error)
	GetVideosByVideoIds(context.Context, uint32, []uint32) ([]*Video, error)
	InitUpdateFavoriteQueue()
	InitUpdateCommentQueue()
}

type PublishUseCase struct {
	repo PublishRepo
	log  *log.Helper
}

func NewPublishUseCase(repo PublishRepo, logger log.Logger) *PublishUseCase {
	go repo.InitUpdateCommentQueue()
	go repo.InitUpdateFavoriteQueue()
	return &PublishUseCase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "model", "usecase/publish")),
	}
}

func (u *PublishUseCase) GetPublishList(
	ctx context.Context, userId uint32,
) ([]*Video, error) {
	if userId == 0 {
		userID := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
		video, err := u.repo.GetVideosByUserId(ctx, userID)
		if err != nil {
			u.log.Errorf("GetPublishList error: %v", err)
		}
		return video, err
	}
	video, err := u.repo.GetVideosByUserId(ctx, userId)
	if err != nil {
		u.log.Errorf("GetPublishList error: %v", err)
	}
	return video, err
}

func (u *PublishUseCase) PublishAction(
	ctx context.Context, fileBytes []byte, title string,
) error {
	err := u.repo.UploadAll(ctx, fileBytes, title)
	if err != nil {
		u.log.Errorf("PublishAction error: %v", err)
	}
	return err
}

func (u *PublishUseCase) GetVideoListByVideoIds(ctx context.Context, userId uint32, videoIds []uint32) ([]*Video, error) {
	video, err := u.repo.GetVideosByVideoIds(ctx, userId, videoIds)
	if err != nil {
		u.log.Errorf("GetVideoListByVideoIds error: %v", err)
	}
	return video, err
}

func (u *PublishUseCase) FeedList(ctx context.Context, latestTime string) (int64, []*Video, error) {
	n, video, err := u.repo.GetFeedList(ctx, latestTime)
	if err != nil {
		u.log.Errorf("FeedList error: %v", err)
	}
	return n, video, err
}
