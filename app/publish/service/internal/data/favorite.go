package data

import (
	"context"
	"errors"

	pb "github.com/toomanysource/atreus/api/favorite/service/v1"
)

type favoriteRepo struct {
	client pb.FavoriteServiceClient
}

func NewFavoriteRepo(conn pb.FavoriteServiceClient) FavoriteRepo {
	return &favoriteRepo{
		client: conn,
	}
}

// IsFavorite 接收favorite服务的回应
func (u *favoriteRepo) IsFavorite(ctx context.Context, userId uint32, videoIds []uint32) ([]bool, error) {
	resp, err := u.client.IsFavorite(ctx, &pb.IsFavoriteRequest{UserId: userId, VideoIds: videoIds})
	if err != nil {
		return nil, errors.Join(ErrFavoriteServiceResponse, err)
	}
	return resp.IsFavorite, nil
}
