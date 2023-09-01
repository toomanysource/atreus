package data

import (
	"context"

	pb "github.com/toomanysource/atreus/api/relation/service/v1"
	"github.com/toomanysource/atreus/app/user/service/internal/server"
)

type favoriteRepo struct {
	client pb.RelationServiceClient
}

func NewRelationRepo(conn server.RelationConn) RelationRepo {
	return &favoriteRepo{
		client: pb.NewRelationServiceClient(conn),
	}
}

// IsFollow 接收Relation服务的回应
func (u *favoriteRepo) IsFollow(ctx context.Context, userId uint32, userIds []uint32) ([]bool, error) {
	resp, err := u.client.IsFollow(ctx, &pb.IsFollowRequest{UserId: userId, ToUserId: userIds})
	if err != nil {
		return nil, err
	}
	return resp.IsFollow, nil
}
