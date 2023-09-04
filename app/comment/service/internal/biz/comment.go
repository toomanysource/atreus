package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/toomanysource/atreus/pkg/errorX"
)

const (
	CreateType uint32 = 1
	DeleteType uint32 = 2
)

type Comment struct {
	Id         uint32
	User       *User
	Content    string
	CreateDate string
}

type User struct {
	Id              uint32
	Name            string
	Avatar          string
	BackgroundImage string
	Signature       string
	IsFollow        bool
	FollowCount     uint32
	FollowerCount   uint32
	TotalFavorited  uint32
	WorkCount       uint32
	FavoriteCount   uint32
}

type CommentRepo interface {
	CreateComment(context.Context, uint32, string) (*Comment, error)
	DeleteComment(context.Context, uint32, uint32) (*Comment, error)
	GetComments(context.Context, uint32) ([]*Comment, error)
}

type CommentUseCase struct {
	repo CommentRepo
	log  *log.Helper
}

func NewCommentUseCase(cr CommentRepo, logger log.Logger) *CommentUseCase {
	return &CommentUseCase{
		repo: cr, log: log.NewHelper(log.With(logger, "model", "usecase/comment")),
	}
}

func (uc *CommentUseCase) GetCommentList(
	ctx context.Context, videoId uint32,
) ([]*Comment, error) {
	return uc.repo.GetComments(ctx, videoId)
}

func (uc *CommentUseCase) CommentAction(
	ctx context.Context, videoId, commentId uint32,
	actionType uint32, commentText string,
) (*Comment, error) {
	switch actionType {
	case CreateType:
		if commentText == "" {
			return nil, errorX.ErrCommentNil
		}
		return uc.repo.CreateComment(ctx, videoId, commentText)
	case DeleteType:
		return uc.repo.DeleteComment(ctx, videoId, commentId)
	default:
		return nil, errorX.ErrInValidActionType
	}
}
