package data

import (
	"context"
	"errors"
	"fmt"

	"github.com/jinzhu/copier"

	pb "github.com/toomanysource/atreus/api/user/service/v1"
	"github.com/toomanysource/atreus/app/publish/service/internal/biz"
	"github.com/toomanysource/atreus/app/publish/service/internal/server"
)

type userRepo struct {
	client pb.UserServiceClient
}

func NewUserRepo(conn server.UserConn) UserRepo {
	return &userRepo{
		client: pb.NewUserServiceClient(conn),
	}
}

// GetUserInfos 接收User服务的回应，并转化为biz.User类型
func (u *userRepo) GetUserInfos(ctx context.Context, userId uint32, userIds []uint32) ([]*biz.User, error) {
	resp, err := u.client.GetUserInfos(ctx, &pb.UserInfosRequest{UserId: userId, UserIds: userIds})
	if err != nil {
		return nil, fmt.Errorf("rpc GetUserInfos error: %v", err)
	}

	// 判空
	if len(resp.Users) == 0 {
		return nil, errors.New("the user service did not search for any information")
	}
	users := make([]*biz.User, 0, len(resp.Users))
	err = copier.Copy(&users, &resp.Users)
	if err != nil {
		return nil, fmt.Errorf("copier.Copy error: %v", err)
	}
	return users, nil
}
