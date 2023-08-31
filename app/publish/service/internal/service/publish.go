package service

import (
	"context"

	"github.com/jinzhu/copier"

	"github.com/toomanysource/atreus/app/publish/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"

	pb "github.com/toomanysource/atreus/api/publish/service/v1"
)

type PublishService struct {
	pb.UnimplementedPublishServiceServer
	log *log.Helper

	usecase *biz.PublishUsecase
}

func NewPublishService(uc *biz.PublishUsecase, logger log.Logger) *PublishService {
	return &PublishService{usecase: uc, log: log.NewHelper(logger)}
}

func (s *PublishService) PublishAction(ctx context.Context, req *pb.PublishActionRequest) (*pb.PublishActionReply, error) {
	err := s.usecase.PublishAction(ctx, req.Data, req.Title)
	if err != nil {
		return &pb.PublishActionReply{
			StatusCode: -1,
			StatusMsg:  err.Error(),
		}, nil
	}
	return &pb.PublishActionReply{
		StatusCode: 0,
		StatusMsg:  "Video published.",
	}, nil
}

func (s *PublishService) GetPublishList(ctx context.Context, req *pb.PublishListRequest) (*pb.PublishListReply, error) {
	reply := &pb.PublishListReply{StatusCode: 0, StatusMsg: "Success", VideoList: make([]*pb.Video, 0)}
	videoList, err := s.usecase.GetPublishList(ctx, req.UserId)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
	}
	err = copier.CopyWithOption(&reply.VideoList, &videoList, copier.Option{DeepCopy: true})
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
	}
	return reply, nil
}

func (s *PublishService) GetVideoListByVideoIds(ctx context.Context, req *pb.VideoListByVideoIdsRequest) (*pb.VideoListReply, error) {
	reply := &pb.VideoListReply{VideoList: make([]*pb.Video, 0)}
	videoList, err := s.usecase.GetVideoListByVideoIds(ctx, req.UserId, req.VideoIds)
	if err != nil {
		return nil, err
	}
	if err = copier.CopyWithOption(&reply.VideoList, &videoList, copier.Option{DeepCopy: true}); err != nil {
		return nil, err
	}
	return reply, nil
}

// FeedList 返回一个按照投稿时间倒序的视频列表，单次最多30个视频
func (s *PublishService) FeedList(ctx context.Context, req *pb.ListFeedRequest) (*pb.ListFeedReply, error) {
	reply := &pb.ListFeedReply{StatusCode: 0, StatusMsg: "Success", VideoList: make([]*pb.Video, 0)}
	nextTime, videos, err := s.usecase.FeedList(ctx, req.LatestTime)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	err = copier.CopyWithOption(&reply.VideoList, &videos, copier.Option{DeepCopy: true})
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	reply.NextTime = nextTime
	return reply, nil
}
