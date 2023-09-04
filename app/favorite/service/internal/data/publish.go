package data

import (
	"context"
	"errors"

	"github.com/toomanysource/atreus/pkg/errorX"

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
	resp, err := f.client.GetVideoListByVideoIds(
		ctx, &pb.VideoListByVideoIdsRequest{UserId: userId, VideoIds: videoIds})
	if err != nil {
		return nil, errors.Join(errorX.ErrPublishServiceResponse, err)
	}

	videos := make([]biz.Video, 0, len(resp.VideoList))
	if err = copier.Copy(&videos, &resp.VideoList); err != nil {
		return nil, errors.Join(errorX.ErrCopy, err)
	}

	return videos, nil
}
