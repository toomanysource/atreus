package service

import (
	"context"

	pb "github.com/toomanysource/atreus/api/relation/service/v1"
	"github.com/toomanysource/atreus/app/relation/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type RelationService struct {
	pb.UnimplementedRelationServiceServer
	log     *log.Helper
	usecase *biz.RelationUsecase
}

func NewRelationService(uc *biz.RelationUsecase, logger log.Logger) *RelationService {
	return &RelationService{usecase: uc, log: log.NewHelper(logger)}
}

// RelationAction 关注/取消关注
func (s *RelationService) RelationAction(ctx context.Context, req *pb.RelationActionRequest) (*pb.RelationActionReply, error) {
	err := s.usecase.Action(ctx, req.ToUserId, req.ActionType)
	if err != nil {
		return &pb.RelationActionReply{
			StatusCode: -1,
			StatusMsg:  err.Error(),
		}, nil
	}
	return &pb.RelationActionReply{
		StatusCode: 0,
		StatusMsg:  "success",
	}, nil
}

// GetFollowRelationList 获取关注列表
func (s *RelationService) GetFollowRelationList(ctx context.Context, req *pb.RelationFollowListRequest) (*pb.RelationFollowListReply, error) {
	reply := &pb.RelationFollowListReply{StatusCode: 0, StatusMsg: "Success"}
	list, err := s.usecase.GetFollowList(ctx, req.UserId)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	for _, user := range list {
		reply.UserList = append(reply.UserList, &pb.User{
			Id:              user.Id,
			Name:            user.Name,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			IsFollow:        user.IsFollow,
			FollowCount:     user.FollowCount,
			FollowerCount:   user.FollowerCount,
			TotalFavorited:  user.TotalFavorite,
			WorkCount:       user.WorkCount,
			FavoriteCount:   user.FavoriteCount,
		})
	}
	return reply, nil
}

// GetFollowerRelationList 获取粉丝列表
func (s *RelationService) GetFollowerRelationList(ctx context.Context, req *pb.RelationFollowerListRequest) (*pb.RelationFollowerListReply, error) {
	reply := &pb.RelationFollowerListReply{StatusCode: 0, StatusMsg: "Success"}
	list, err := s.usecase.GetFollowerList(ctx, req.UserId)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	for _, user := range list {
		reply.UserList = append(reply.UserList, &pb.User{
			Id:              user.Id,
			Name:            user.Name,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			IsFollow:        user.IsFollow,
			FollowCount:     user.FollowCount,
			FollowerCount:   user.FollowerCount,
			TotalFavorited:  user.TotalFavorite,
			WorkCount:       user.WorkCount,
			FavoriteCount:   user.FavoriteCount,
		})
	}
	return reply, nil
}

// GetFriendRelationList 获取粉丝列表
func (s *RelationService) GetFriendRelationList(ctx context.Context, req *pb.RelationFriendListRequest) (*pb.RelationFriendListReply, error) {
	reply := &pb.RelationFriendListReply{StatusCode: 0, StatusMsg: "Success"}
	list, err := s.usecase.GetFollowerList(ctx, req.UserId)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	for _, user := range list {
		reply.UserList = append(reply.UserList, &pb.FriendUser{
			Id:              user.Id,
			Name:            user.Name,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			IsFollow:        user.IsFollow,
			FollowCount:     user.FollowCount,
			FollowerCount:   user.FollowerCount,
			TotalFavorited:  user.TotalFavorite,
			WorkCount:       user.WorkCount,
			FavoriteCount:   user.FavoriteCount,
		})
	}
	return reply, nil
}

func (s *RelationService) IsFollow(ctx context.Context, req *pb.IsFollowRequest) (*pb.IsFollowReply, error) {
	isFollow, err := s.usecase.IsFollow(ctx, req.UserId, req.ToUserId)
	if err != nil {
		return nil, err
	}
	return &pb.IsFollowReply{
		IsFollow: isFollow,
	}, nil
}
