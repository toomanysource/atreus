package biz

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
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
	GetCommentList(context.Context, uint32) ([]*Comment, error)
}

type CommentUsecase struct {
	commentRepo CommentRepo
	log         *log.Helper
}

func NewCommentUsecase(cr CommentRepo, logger log.Logger) *CommentUsecase {
	return &CommentUsecase{
		commentRepo: cr, log: log.NewHelper(log.With(logger, "model", "usecase/comment")),
	}
}

func (uc *CommentUsecase) GetCommentList(
	ctx context.Context, videoId uint32,
) ([]*Comment, error) {
	return uc.commentRepo.GetCommentList(ctx, videoId)
}

func (uc *CommentUsecase) CommentAction(
	ctx context.Context, videoId, commentId uint32,
	actionType uint32, commentText string,
) (*Comment, error) {
	// 判断actionType是否在指定范围内
	var createType uint32 = 1
	var deleteType uint32 = 2
	switch actionType {
	case createType:
		return uc.commentRepo.CreateComment(ctx, videoId, commentText)
	case deleteType:
		return uc.commentRepo.DeleteComment(ctx, videoId, commentId)
	default:
		return nil, errors.New("the value of action_type is not in the specified range")
	}
}
