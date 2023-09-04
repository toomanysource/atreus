package service

import (
	"context"

	"github.com/toomanysource/atreus/pkg/errorX"

	"github.com/jinzhu/copier"

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
	reply := &pb.FavoriteListReply{StatusCode: errorX.CodeSuccess, StatusMsg: "success", VideoList: make([]*pb.Video, 0)}
	videos, err := s.fu.GetFavoriteList(ctx, req.UserId)
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	err = copier.CopyWithOption(&reply.VideoList, &videos, copier.Option{
		DeepCopy: true,
	})
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}

func (s *FavoriteService) FavoriteAction(
	ctx context.Context, req *pb.FavoriteActionRequest,
) (*pb.FavoriteActionReply, error) {
	reply := &pb.FavoriteActionReply{StatusCode: errorX.CodeSuccess, StatusMsg: "success"}
	err := s.fu.FavoriteAction(ctx, req.VideoId, req.ActionType)
	if err != nil {
		reply.StatusCode = errorX.CodeFailed
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
