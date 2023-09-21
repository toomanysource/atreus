package data

import (
	"context"
	"errors"

	"github.com/jinzhu/copier"

	pb "github.com/toomanysource/atreus/api/user/service/v1"
	"github.com/toomanysource/atreus/app/relation/service/internal/biz"
)

type userRepo struct {
	client pb.UserServiceClient
}

func NewUserRepo(conn pb.UserServiceClient) UserRepo {
	return &userRepo{
		client: conn,
	}
}

// GetUserInfos 接收User服务的回应，并转化为biz.User类型
func (u *userRepo) GetUserInfos(ctx context.Context, userId uint32, userIds []uint32) ([]*biz.User, error) {
	resp, err := u.client.GetUserInfos(ctx, &pb.UserInfosRequest{UserId: userId, UserIds: userIds})
	if err != nil {
		return nil, errors.Join(ErrUserServiceResponse, err)
	}

	users := make([]*biz.User, 0, len(resp.Users))
	if err = copier.Copy(&users, &resp.Users); err != nil {
		return nil, errors.Join(ErrCopy, err)
	}
	return users, nil
}
