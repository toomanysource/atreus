package service

import (
	"context"

	"github.com/jinzhu/copier"

	"github.com/toomanysource/atreus/app/user/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/toomanysource/atreus/api/user/service/v1"
)

const (
	CodeSuccess = 0
	CodeFailed  = 300
)

type UserService struct {
	pb.UnimplementedUserServiceServer

	log *log.Helper

	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase, logger log.Logger) *UserService {
	return &UserService{uc: uc, log: log.NewHelper(logger)}
}

func (s *UserService) UserRegister(ctx context.Context, req *pb.UserRegisterRequest) (*pb.UserRegisterReply, error) {
	user, err := s.uc.Register(ctx, req.Username, req.Password)
	if err != nil {
		return &pb.UserRegisterReply{
			StatusCode: CodeFailed,
			StatusMsg:  err.Error(),
		}, nil
	}
	reply := &pb.UserRegisterReply{
		StatusCode: CodeSuccess,
		StatusMsg:  "success",
	}
	copier.Copy(reply, user)
	return reply, nil
}

func (s *UserService) UserLogin(ctx context.Context, req *pb.UserLoginRequest) (*pb.UserLoginReply, error) {
	user, err := s.uc.Login(ctx, req.Username, req.Password)
	if err != nil {
		return &pb.UserLoginReply{
			StatusCode: CodeFailed,
			StatusMsg:  err.Error(),
		}, nil
	}
	reply := &pb.UserLoginReply{
		StatusCode: CodeSuccess,
		StatusMsg:  "success",
	}
	copier.Copy(reply, user)
	return reply, nil
}

func (s *UserService) GetUserInfo(ctx context.Context, req *pb.UserInfoRequest) (*pb.UserInfoReply, error) {
	user, err := s.uc.GetInfo(ctx, req.UserId)
	if err != nil {
		return &pb.UserInfoReply{
			StatusCode: CodeFailed,
			StatusMsg:  err.Error(),
		}, nil
	}
	reply := &pb.UserInfoReply{
		StatusCode: CodeSuccess,
		StatusMsg:  "success",
		User:       new(pb.User),
	}
	copier.Copy(reply.User, user)
	return reply, nil
}

func (s *UserService) GetUserInfos(ctx context.Context, req *pb.UserInfosRequest) (*pb.UserInfosReply, error) {
	users, err := s.uc.GetInfos(ctx, req.UserId, req.UserIds)
	if err != nil {
		return nil, err
	}
	reply := &pb.UserInfosReply{
		Users: make([]*pb.User, len(users)),
	}
	copier.Copy(&reply.Users, &users)
	return reply, nil
}

func (s *UserService) UpdateFollow(ctx context.Context, req *pb.UpdateFollowRequest) (*emptypb.Empty, error) {
	err := s.uc.UpdateFollow(ctx, req.UserId, req.FollowChange)
	return &emptypb.Empty{}, err
}

func (s *UserService) UpdateFollower(ctx context.Context, req *pb.UpdateFollowerRequest) (*emptypb.Empty, error) {
	err := s.uc.UpdateFollower(ctx, req.UserId, req.FollowerChange)
	return &emptypb.Empty{}, err
}

func (s *UserService) UpdateFavorited(ctx context.Context, req *pb.UpdateFavoritedRequest) (*emptypb.Empty, error) {
	err := s.uc.UpdateFavorited(ctx, req.UserId, req.FavoritedChange)
	return &emptypb.Empty{}, err
}

func (s *UserService) UpdateWork(ctx context.Context, req *pb.UpdateWorkRequest) (*emptypb.Empty, error) {
	err := s.uc.UpdateWork(ctx, req.UserId, req.WorkChange)
	return &emptypb.Empty{}, err
}

func (s *UserService) UpdateFavorite(ctx context.Context, req *pb.UpdateFavoriteRequest) (*emptypb.Empty, error) {
	err := s.uc.UpdateFavorite(ctx, req.UserId, req.FavoriteChange)
	return &emptypb.Empty{}, err
}
