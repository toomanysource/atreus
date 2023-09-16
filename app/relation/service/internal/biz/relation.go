package biz

import (
	"context"

	"github.com/toomanysource/atreus/middleware"

	"github.com/go-kratos/kratos/v2/log"
)

type User struct {
	Id              uint32 // 用户id
	Name            string // 用户名称
	FollowCount     uint32 // 关注总数
	FollowerCount   uint32 // 粉丝总数
	IsFollow        bool   // true-已关注，false-未关注
	Avatar          string // 用户头像
	BackgroundImage string // 用户个人页顶部大图
	Signature       string // 个人简介
	TotalFavorite   uint32 // 获赞数量
	WorkCount       uint32 // 作品数量
	FavoriteCount   uint32 // 点赞数量
}

type RelationRepo interface {
	GetFollowList(context.Context, uint32) ([]*User, error)
	GetFollowerList(context.Context, uint32) ([]*User, error)
	Follow(context.Context, uint32) error
	UnFollow(context.Context, uint32) error
	IsFollow(ctx context.Context, userId uint32, toUserId []uint32) ([]bool, error)
}

type RelationUseCase struct {
	repo RelationRepo
	log  *log.Helper
}

func NewRelationUseCase(repo RelationRepo, logger log.Logger) *RelationUseCase {
	return &RelationUseCase{repo: repo, log: log.NewHelper(logger)}
}

// GetFollowList 获取关注列表
func (uc *RelationUseCase) GetFollowList(ctx context.Context, userId uint32) ([]*User, error) {
	if userId == 0 {
		userID := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
		users, err := uc.repo.GetFollowList(ctx, userID)
		if err != nil {
			uc.log.Errorf("GetFollowList error: %v", err)
		}
		return users, err
	}
	users, err := uc.repo.GetFollowList(ctx, userId)
	if err != nil {
		uc.log.Errorf("GetFollowList error: %v", err)
	}
	return users, err
}

// GetFollowerList 获取粉丝列表
func (uc *RelationUseCase) GetFollowerList(ctx context.Context, userId uint32) ([]*User, error) {
	if userId == 0 {
		userID := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
		users, err := uc.repo.GetFollowerList(ctx, userID)
		if err != nil {
			uc.log.Errorf("GetFollowerList error: %v", err)
		}
		return users, err
	}
	users, err := uc.repo.GetFollowerList(ctx, userId)
	if err != nil {
		uc.log.Errorf("GetFollowerList error: %v", err)
	}
	return users, err
}

// Action 关注和取消关注
func (uc *RelationUseCase) Action(ctx context.Context, toUserId uint32, actionType uint32) (err error) {
	switch actionType {
	// 1为关注
	case FollowType:
		if err = uc.repo.Follow(ctx, toUserId); err != nil {
			uc.log.Errorf("Follow error: %v", err)
		}
		return err
	// 2为取消关注
	case UnfollowType:
		if err = uc.repo.UnFollow(ctx, toUserId); err != nil {
			uc.log.Errorf("UnFollow error: %v", err)
		}
		return err
	default:
		return ErrInValidActionType
	}
}

func (uc *RelationUseCase) IsFollow(ctx context.Context, userId uint32, toUserId []uint32) ([]bool, error) {
	oks, err := uc.repo.IsFollow(ctx, userId, toUserId)
	if err != nil {
		uc.log.Errorf("IsFollow error: %v", err)
	}
	return oks, err
}
