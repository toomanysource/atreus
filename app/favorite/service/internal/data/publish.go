package data

import (
	"context"

	"github.com/toomanysource/atreus/app/favorite/service/internal/biz"
	"github.com/toomanysource/atreus/app/favorite/service/internal/server"

	pb "github.com/toomanysource/atreus/api/publish/service/v1"

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

// GetVideoListByVideoIds 通过videoId获取视频信息;
func (f *publishRepo) GetVideoListByVideoIds(
	ctx context.Context, userId uint32, videoIds []uint32,
) ([]biz.Video, error) {
	// call grpc function to fetch video info
	resp, err := f.client.GetVideoListByVideoIds(ctx, &pb.VideoListByVideoIdsRequest{UserId: userId, VideoIds: videoIds})
	if err != nil {
		return nil, err
	}
	// convert pb.Video slice to biz.Video slice
	videos := make([]biz.Video, len(resp.VideoList))
	if err := copier.Copy(&videos, &resp.VideoList); err != nil {
		return nil, err
	}
	return videos, nil
}
