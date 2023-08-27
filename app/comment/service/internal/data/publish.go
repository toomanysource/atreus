package data

import (
	"context"

	pb "github.com/toomanysource/atreus/api/publish/service/v1"
	"github.com/toomanysource/atreus/app/comment/service/internal/server"
)

type publishRepo struct {
	client pb.PublishServiceClient
}

func NewPublishRepo(conn server.PublishConn) PublishRepo {
	return &publishRepo{
		client: pb.NewPublishServiceClient(conn),
	}
}

// UpdateComment 接收Publish服务的回应
func (u *publishRepo) UpdateComment(ctx context.Context, videoId uint32, commentChange int32) error {
	_, err := u.client.UpdateComment(
		ctx, &pb.UpdateCommentCountRequest{VideoId: videoId, CommentChange: commentChange})
	return err
}
