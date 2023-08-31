package service

import (
	"context"

	"github.com/jinzhu/copier"

	"github.com/toomanysource/atreus/app/user/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"

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
		UserId:     user.Id,
		Token:      user.Token,
	}
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
		UserId:     user.Id,
		Token:      user.Token,
	}
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
