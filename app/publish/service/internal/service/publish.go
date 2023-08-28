package service

import (
	"context"
	"fmt"

	"github.com/toomanysource/atreus/app/publish/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/emptypb"

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
	videoList, err := s.usecase.GetPublishList(ctx, req.UserId)
	pbVideoList := bizVideoList2pbVideoList(videoList)
	if err != nil {
		return &pb.PublishListReply{
			StatusCode: -1,
			StatusMsg:  err.Error(),
			VideoList:  nil,
		}, nil
	}
	return &pb.PublishListReply{
		StatusCode: 0,
		StatusMsg:  "Return video list.",
		VideoList:  pbVideoList,
	}, nil
}

func (s *PublishService) GetVideoList(ctx context.Context, req *pb.VideoListRequest) (*pb.VideoListReply, error) {
	nextTime, videoList, err := s.usecase.GetVideoList(ctx, req.LatestTime, req.UserId, req.Number)
	if err != nil {
		return nil, fmt.Errorf("rpc GetVideoList error: %v", err)
	}
	pbVideoList := bizVideoList2pbVideoList(videoList)
	return &pb.VideoListReply{
		NextTime:  nextTime,
		VideoList: pbVideoList,
	}, nil
}

func (s *PublishService) GetVideoListByVideoIds(ctx context.Context, req *pb.VideoListByVideoIdsRequest) (*pb.VideoListReply, error) {
	videoList, err := s.usecase.GetVideoListByVideoIds(ctx, req.UserId, req.VideoIds)
	if err != nil {
		return nil, err
	}
	pbVideoList := bizVideoList2pbVideoList(videoList)
	return &pb.VideoListReply{
		VideoList: pbVideoList,
	}, nil
}

func (s *PublishService) UpdateComment(ctx context.Context, req *pb.UpdateCommentCountRequest) (*emptypb.Empty, error) {
	err := s.usecase.UpdateComment(ctx, req.VideoId, req.CommentChange)
	return &emptypb.Empty{}, err
}

func (s *PublishService) UpdateFavorite(ctx context.Context, req *pb.UpdateFavoriteCountRequest) (*emptypb.Empty, error) {
	err := s.usecase.UpdateFavorite(ctx, req.VideoId, req.FavoriteChange)
	return &emptypb.Empty{}, err
}

func bizVideoList2pbVideoList(bizVideoList []*biz.Video) (pbVideoList []*pb.Video) {
	for _, video := range bizVideoList {
		pbVideoList = append(pbVideoList, &pb.Video{
			Id: video.ID,
			Author: &pb.User{
				Id:              video.Author.ID,
				Name:            video.Author.Name,
				FollowCount:     video.Author.FollowCount,
				FollowerCount:   video.Author.FollowerCount,
				IsFollow:        video.Author.IsFollow,
				Avatar:          video.Author.Avatar,
				BackgroundImage: video.Author.BackgroundImage,
				Signature:       video.Author.Signature,
				TotalFavorited:  video.Author.TotalFavorited,
				WorkCount:       video.Author.WorkCount,
				FavoriteCount:   video.Author.FavoriteCount,
			},
			PlayUrl:       video.PlayUrl,
			CoverUrl:      video.CoverUrl,
			FavoriteCount: video.FavoriteCount,
			CommentCount:  video.CommentCount,
			IsFavorite:    video.IsFavorite,
			Title:         video.Title,
		})
	}

	return pbVideoList
}
