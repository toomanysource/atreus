package biz

import (
	"context"

	"github.com/toomanysource/atreus/app/publish/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

// Video is a video model
type Video struct {
	ID            uint32
	Author        *User
	PlayUrl       string
	CoverUrl      string
	FavoriteCount uint32
	CommentCount  uint32
	IsFavorite    bool
	Title         string
}

// User is a user model.
type User struct {
	ID              uint32
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

// PublishRepo is a publishing repo.
type PublishRepo interface {
	FindVideoListByUserId(context.Context, uint32) ([]*Video, error)
	UploadVideo(context.Context, []byte, string) error
	GetFeedList(context.Context, string) (int64, []*Video, error)
	FindVideoListByVideoIds(context.Context, uint32, []uint32) ([]*Video, error)
	InitUpdateFavoriteQueue()
	InitUpdateCommentQueue()
}

// PublishUsecase is a publishing usecase.
type PublishUsecase struct {
	repo   PublishRepo
	config *conf.JWT
	log    *log.Helper
}

// NewPublishUsecase new a publishing usecase.
func NewPublishUsecase(repo PublishRepo, JWTConf *conf.JWT, logger log.Logger) *PublishUsecase {
	go repo.InitUpdateCommentQueue()
	go repo.InitUpdateFavoriteQueue()
	return &PublishUsecase{repo: repo, config: JWTConf, log: log.NewHelper(logger)}
}

func (u *PublishUsecase) GetPublishList(
	ctx context.Context, userId uint32,
) ([]*Video, error) {
	return u.repo.FindVideoListByUserId(ctx, userId)
}

func (u *PublishUsecase) PublishAction(
	ctx context.Context, fileBytes []byte, title string,
) error {
	return u.repo.UploadVideo(ctx, fileBytes, title)
}

func (u *PublishUsecase) GetVideoListByVideoIds(ctx context.Context, userId uint32, videoIds []uint32) ([]*Video, error) {
	videoList, err := u.repo.FindVideoListByVideoIds(ctx, userId, videoIds)
	return videoList, err
}

func (u *PublishUsecase) FeedList(ctx context.Context, latestTime string) (int64, []*Video, error) {
	return u.repo.GetFeedList(ctx, latestTime)
}
