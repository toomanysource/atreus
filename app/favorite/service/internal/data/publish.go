package data

import (
	"context"
	"errors"

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
	if len(resp.VideoList) == 0 {
		return nil, errors.New("video not found")
	}
	// convert pb.Video slice to biz.Video slice
	videos := make([]biz.Video, len(resp.VideoList))
	if err := copier.Copy(&videos, &resp.VideoList); err != nil {
		return nil, err
	}
	return videos, nil
}

// UpdateFavoriteCount 更新视频点赞数 - 在点赞/取消点赞时调用
func (f *publishRepo) UpdateFavoriteCount(ctx context.Context, videoId uint32, change int32) error {
	_, err := f.client.UpdateFavorite(ctx, &pb.UpdateFavoriteCountRequest{
		VideoId:        videoId,
		FavoriteChange: change,
	})
	if err != nil {
		return err
	}
	return nil
}
