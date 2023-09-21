package data

import (
	"context"
	"errors"

	pb "github.com/toomanysource/atreus/api/publish/service/v1"
	"github.com/toomanysource/atreus/app/favorite/service/internal/biz"

	"github.com/jinzhu/copier"
)

type publishRepo struct {
	client pb.PublishServiceClient
}

func NewPublishRepo(conn pb.PublishServiceClient) biz.PublishRepo {
	return &publishRepo{
		client: conn,
	}
}

// GetVideoListByVideoIds 通过videoId获取视频信息;
func (f *publishRepo) GetVideoListByVideoIds(
	ctx context.Context, userId uint32, videoIds []uint32,
) ([]biz.Video, error) {
	resp, err := f.client.GetVideoListByVideoIds(
		ctx, &pb.VideoListByVideoIdsRequest{UserId: userId, VideoIds: videoIds})
	if err != nil {
		return nil, errors.Join(ErrPublishServiceResponse, err)
	}

	videos := make([]biz.Video, 0, len(resp.VideoList))
	if err = copier.Copy(&videos, &resp.VideoList); err != nil {
		return nil, errors.Join(ErrCopy, err)
	}

	return videos, nil
}
