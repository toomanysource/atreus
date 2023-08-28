package data

import (
	"context"

	"github.com/toomanysource/atreus/app/favorite/service/internal/biz"
	"github.com/toomanysource/atreus/app/favorite/service/internal/server"

	pb "github.com/toomanysource/atreus/api/user/service/v1"
)

type userRepo struct {
	client pb.UserServiceClient
}

func NewUserRepo(conn server.UserConn) biz.UserRepo {
	return &userRepo{
		client: pb.NewUserServiceClient(conn),
	}
}

func (u *userRepo) UpdateFavorited(ctx context.Context, userId uint32, change int32) error {
	_, err := u.client.UpdateFavorited(ctx, &pb.UpdateFavoritedRequest{
		UserId:          userId,
		FavoritedChange: change,
	})
	if err != nil {
		return err
	}
	return nil
}

func (u *userRepo) UpdateFavorite(ctx context.Context, userId uint32, change int32) error {
	_, err := u.client.UpdateFavorite(ctx, &pb.UpdateFavoriteRequest{
		UserId:         userId,
		FavoriteChange: change,
	})
	if err != nil {
		return err
	}
	return nil
}
