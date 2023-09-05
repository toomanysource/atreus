package data

import (
	"context"
	"errors"

	"github.com/toomanysource/atreus/pkg/errorX"

	pb "github.com/toomanysource/atreus/api/favorite/service/v1"
	"github.com/toomanysource/atreus/app/publish/service/internal/server"
)

type favoriteRepo struct {
	client pb.FavoriteServiceClient
}

func NewFavoriteRepo(conn server.FavoriteConn) FavoriteRepo {
	return &favoriteRepo{
		client: pb.NewFavoriteServiceClient(conn),
	}
}

// IsFavorite 接收favorite服务的回应
func (u *favoriteRepo) IsFavorite(ctx context.Context, userId uint32, videoIds []uint32) ([]bool, error) {
	resp, err := u.client.IsFavorite(ctx, &pb.IsFavoriteRequest{UserId: userId, VideoIds: videoIds})
	if err != nil {
		return nil, errors.Join(errorX.ErrFavoriteServiceResponse, err)
	}
	return resp.IsFavorite, nil
}
