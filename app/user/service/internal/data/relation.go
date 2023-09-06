package data

import (
	"context"

	"github.com/toomanysource/atreus/app/user/service/internal/biz"

	pb "github.com/toomanysource/atreus/api/relation/service/v1"
	"github.com/toomanysource/atreus/app/user/service/internal/server"
)

type relationRepo struct {
	client pb.RelationServiceClient
}

func NewRelationRepo(conn server.RelationConn) biz.RelationRepo {
	return &relationRepo{
		client: pb.NewRelationServiceClient(conn),
	}
}

// IsFollow 接收Relation服务的回应
func (u *relationRepo) IsFollow(ctx context.Context, userId uint32, userIds []uint32) ([]bool, error) {
	resp, err := u.client.IsFollow(ctx, &pb.IsFollowRequest{UserId: userId, ToUserId: userIds})
	if err != nil {
		return nil, err
	}
	return resp.IsFollow, nil
}
