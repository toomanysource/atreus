package data

import (
	"context"
	"errors"

	"github.com/jinzhu/copier"

	"github.com/toomanysource/atreus/pkg/errorX"

	pb "github.com/toomanysource/atreus/api/user/service/v1"
	"github.com/toomanysource/atreus/app/comment/service/internal/biz"
	"github.com/toomanysource/atreus/app/comment/service/internal/server"
)

type userRepo struct {
	client pb.UserServiceClient
}

func NewUserRepo(conn server.UserConn) biz.UserRepo {
	return &userRepo{
		client: pb.NewUserServiceClient(conn),
	}
}

// GetUserInfos 接收User服务的回应，并转化为biz.User类型
func (u *userRepo) GetUserInfos(ctx context.Context, userId uint32, userIds []uint32) ([]*biz.User, error) {
	resp, err := u.client.GetUserInfos(ctx, &pb.UserInfosRequest{UserId: userId, UserIds: userIds})
	if err != nil {
		return nil, errors.Join(errorX.ErrUserServiceResponse, err)
	}

	users := make([]*biz.User, 0, len(resp.Users))
	if err = copier.Copy(&users, &resp.Users); err != nil {
		return nil, errors.Join(errorX.ErrCopy, err)
	}
	return users, nil
}
