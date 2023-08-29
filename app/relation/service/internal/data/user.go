package data

import (
	"context"

	pb "github.com/toomanysource/atreus/api/user/service/v1"
	"github.com/toomanysource/atreus/app/relation/service/internal/biz"

	"google.golang.org/grpc"
)

type userRepo struct {
	client pb.UserServiceClient
}

func NewUserRepo(conn *grpc.ClientConn) UserRepo {
	return &userRepo{
		client: pb.NewUserServiceClient(conn),
	}
}

// GetUserInfos 接收User服务的回应，并转化为biz.User类型
func (u *userRepo) GetUserInfos(ctx context.Context, userId uint32, userIds []uint32) ([]*biz.User, error) {
	resp, err := u.client.GetUserInfos(ctx, &pb.UserInfosRequest{UserId: userId, UserIds: userIds})
	if err != nil {
		return nil, err
	}

	// 判空
	if len(resp.Users) == 0 {
		return nil, nil
	}

	users := make([]*biz.User, 0, len(resp.Users)+1)
	for _, user := range resp.Users {
		users = append(users, &biz.User{
			Id:              user.Id,
			Name:            user.Name,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			IsFollow:        user.IsFollow,
			FollowCount:     user.FollowCount,
			FollowerCount:   user.FollowerCount,
			TotalFavorite:   user.TotalFavorited,
			WorkCount:       user.WorkCount,
			FavoriteCount:   user.FavoriteCount,
		})
	}
	return users, nil
}
