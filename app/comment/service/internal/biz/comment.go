package biz

import (
	"context"
	"time"

	"github.com/toomanysource/atreus/middleware"

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
	CreateComment(ctx context.Context, userId, videoId uint32, commentText string, createTime string) (*Comment, error)
	DeleteComment(ctx context.Context, userId, videoId, commentId uint32) error
	GetComments(ctx context.Context, videoId uint32) (cls []*Comment, err error)
}

type UserRepo interface {
	GetUserInfos(context.Context, uint32, []uint32) ([]*User, error)
}

type CommentUseCase struct {
	repo     CommentRepo
	userRepo UserRepo
	log      *log.Helper
}

func NewCommentUseCase(cr CommentRepo, user UserRepo, logger log.Logger) *CommentUseCase {
	return &CommentUseCase{
		repo:     cr,
		userRepo: user,
		log:      log.NewHelper(log.With(logger, "model", "usecase/comment")),
	}
}

func (uc *CommentUseCase) GetCommentList(
	ctx context.Context, videoId uint32,
) ([]*Comment, error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	cls, err := uc.repo.GetComments(ctx, videoId)
	if err != nil {
		uc.log.Errorf("%v: %v", ErrGetCommentList, err)
		return nil, ErrGetCommentList
	}
	// 获取评论列表中的所有用户id
	userIds := make([]uint32, 0, len(cls))
	for _, comment := range cls {
		userIds = append(userIds, comment.User.Id)
	}

	// 统一查询，减少网络IO
	users, err := uc.userRepo.GetUserInfos(ctx, userId, userIds)
	if err != nil {
		uc.log.Errorf("%v: %v", ErrGetCommentList, err)
		return nil, ErrGetCommentList
	}
	for i, comment := range cls {
		comment.User = users[i]
	}
	return cls, nil
}

func (uc *CommentUseCase) CommentAction(
	ctx context.Context, videoId, commentId uint32,
	actionType uint32, commentText string,
) (*Comment, error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	switch actionType {
	case CreateType:
		if commentText == "" {
			return nil, ErrCommentTextEmpty
		}
		createTime := time.Now().Format("01-02")
		c, err := uc.repo.CreateComment(ctx, userId, videoId, commentText, createTime)
		if err != nil {
			uc.log.Errorf("%v: %v", ErrCreateComment, err)
			return nil, ErrCreateComment
		}
		users, err := uc.userRepo.GetUserInfos(ctx, userId, []uint32{userId})
		if err != nil {
			uc.log.Errorf("%v: %v", ErrCreateComment, err)
			return nil, ErrCreateComment
		}
		c.User = users[0]
		return c, nil
	case DeleteType:
		err := uc.repo.DeleteComment(ctx, userId, videoId, commentId)
		if err != nil {
			uc.log.Errorf("%v: %v", ErrDeleteComment, err)
			return nil, ErrDeleteComment
		}
		return nil, nil
	default:
		return nil, ErrInValidActionType
	}
}
