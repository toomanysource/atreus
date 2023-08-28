package service

import (
	"context"

	"github.com/toomanysource/atreus/app/favorite/service/internal/biz"

	pb "github.com/toomanysource/atreus/api/favorite/service/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type FavoriteService struct {
	pb.UnimplementedFavoriteServiceServer
	fu  *biz.FavoriteUseCase
	log *log.Helper
}

func NewFavoriteService(fu *biz.FavoriteUseCase, logger log.Logger) *FavoriteService {
	return &FavoriteService{
		fu:  fu,
		log: log.NewHelper(log.With(logger, "model", "service/favorite")),
	}
}

func (s *FavoriteService) GetFavoriteList(
	ctx context.Context, req *pb.FavoriteListRequest,
) (*pb.FavoriteListReply, error) {
	reply := &pb.FavoriteListReply{StatusCode: 0, StatusMsg: "success"}
	videos, err := s.fu.GetFavoriteList(ctx, req.UserId)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	for _, video := range videos {
		reply.VideoList = append(reply.VideoList, &pb.Video{
			Id:    video.Id,
			Title: video.Title,
			Author: &pb.User{
				Id:              video.Author.Id,
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
		})
	}
	return reply, nil
}

func (s *FavoriteService) FavoriteAction(
	ctx context.Context, req *pb.FavoriteActionRequest,
) (*pb.FavoriteActionReply, error) {
	reply := &pb.FavoriteActionReply{StatusCode: 0, StatusMsg: "success"}
	err := s.fu.FavoriteAction(ctx, req.VideoId, req.ActionType)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}

func (s *FavoriteService) IsFavorite(
	ctx context.Context, req *pb.IsFavoriteRequest,
) (*pb.IsFavoriteReply, error) {
	isFavorite, err := s.fu.IsFavorite(ctx, req.UserId, req.VideoIds)
	if err != nil {
		return nil, err
	}
	return &pb.IsFavoriteReply{
		IsFavorite: isFavorite,
	}, nil
}
