package data

import (
	"context"
	"fmt"

	pb "github.com/toomanysource/atreus/api/publish/service/v1"
	"github.com/toomanysource/atreus/app/feed/service/internal/biz"
	"github.com/toomanysource/atreus/app/feed/service/internal/server"

	"github.com/jinzhu/copier"
)

type publishRepo struct {
	client pb.PublishServiceClient
}

func NewPublishRepo(conn server.PublishConn) biz.PublishRepo {
	return &publishRepo{
		client: pb.NewPublishServiceClient(conn),
	}
}

func (u *publishRepo) GetVideoList(
	ctx context.Context, latestTime string, userId uint32, number uint32,
) (int64, []*biz.Video, error) {
	resp, err := u.client.GetVideoList(ctx, &pb.VideoListRequest{
		LatestTime: latestTime, UserId: userId, Number: number,
	})
	if err != nil {
		return 0, nil, fmt.Errorf("rpc GetVideoList error: %v", err)
	}

	videos := make([]*biz.Video, len(resp.VideoList))
	if err = copier.Copy(&videos, &resp.VideoList); err != nil {
		return 0, nil, fmt.Errorf("copier.Copy error: %v", err)
	}

	nextTime := resp.NextTime
	return nextTime, videos, nil
}
