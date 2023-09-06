package service

import (
	"context"

	"github.com/jinzhu/copier"

	"github.com/toomanysource/atreus/pkg/errorX"

	pb "github.com/toomanysource/atreus/api/relation/service/v1"
	"github.com/toomanysource/atreus/app/relation/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type RelationService struct {
	pb.UnimplementedRelationServiceServer
	log *log.Helper
	ru  *biz.RelationUseCase
}

func NewRelationService(uc *biz.RelationUseCase, logger log.Logger) *RelationService {
	return &RelationService{
		ru:  uc,
		log: log.NewHelper(log.With(logger, "model", "service/relation")),
	}
}

// RelationAction 关注/取消关注
func (s *RelationService) RelationAction(ctx context.Context, req *pb.RelationActionRequest) (*pb.RelationActionReply, error) {
	reply := &pb.RelationActionReply{StatusCode: errorX.CodeSuccess, StatusMsg: "success"}
	err := s.ru.Action(ctx, req.ToUserId, req.ActionType)
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}

// GetFollowRelationList 获取关注列表
func (s *RelationService) GetFollowRelationList(ctx context.Context, req *pb.RelationFollowListRequest) (*pb.RelationFollowListReply, error) {
	reply := &pb.RelationFollowListReply{StatusCode: errorX.CodeSuccess, StatusMsg: "success", UserList: make([]*pb.User, 0)}
	list, err := s.ru.GetFollowList(ctx, req.UserId)
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	err = copier.Copy(&reply.UserList, &list)
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}

// GetFollowerRelationList 获取粉丝列表
func (s *RelationService) GetFollowerRelationList(ctx context.Context, req *pb.RelationFollowerListRequest) (*pb.RelationFollowerListReply, error) {
	reply := &pb.RelationFollowerListReply{StatusCode: errorX.CodeSuccess, StatusMsg: "success", UserList: make([]*pb.User, 0)}
	list, err := s.ru.GetFollowerList(ctx, req.UserId)
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	err = copier.Copy(&reply.UserList, &list)
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}

// GetFriendRelationList 获取粉丝列表
func (s *RelationService) GetFriendRelationList(ctx context.Context, req *pb.RelationFriendListRequest) (*pb.RelationFriendListReply, error) {
	reply := &pb.RelationFriendListReply{StatusCode: errorX.CodeSuccess, StatusMsg: "success", UserList: make([]*pb.FriendUser, 0)}
	list, err := s.ru.GetFollowerList(ctx, req.UserId)
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	err = copier.Copy(&reply.UserList, &list)
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}

func (s *RelationService) IsFollow(ctx context.Context, req *pb.IsFollowRequest) (*pb.IsFollowReply, error) {
	isFollow, err := s.ru.IsFollow(ctx, req.UserId, req.ToUserId)
	if err != nil {
		return nil, err
	}
	return &pb.IsFollowReply{
		IsFollow: isFollow,
	}, nil
}
