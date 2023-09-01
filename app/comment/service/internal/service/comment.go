package service

import (
	"context"

	"github.com/jinzhu/copier"

	pb "github.com/toomanysource/atreus/api/comment/service/v1"
	"github.com/toomanysource/atreus/app/comment/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type CommentService struct {
	pb.UnimplementedCommentServiceServer
	cu  *biz.CommentUsecase
	log *log.Helper
}

func NewCommentService(cu *biz.CommentUsecase, logger log.Logger) *CommentService {
	return &CommentService{
		cu:  cu,
		log: log.NewHelper(log.With(logger, "model", "service/comment")),
	}
}

func (s *CommentService) GetCommentList(ctx context.Context, req *pb.CommentListRequest) (*pb.CommentListReply, error) {
	reply := &pb.CommentListReply{StatusCode: 0, StatusMsg: "Success", CommentList: make([]*pb.Comment, 0)}
	commentList, err := s.cu.GetCommentList(ctx, req.VideoId)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	err = copier.CopyWithOption(&reply.CommentList, &commentList, copier.Option{DeepCopy: true})
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}

func (s *CommentService) CommentAction(ctx context.Context, req *pb.CommentActionRequest) (*pb.CommentActionReply, error) {
	reply := &pb.CommentActionReply{StatusCode: 0, StatusMsg: "Success", Comment: &pb.Comment{}}
	comment, err := s.cu.CommentAction(ctx, req.VideoId, req.CommentId, req.ActionType, req.CommentText)
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	// 删除功能无comment值
	if comment == nil {
		return reply, nil
	}
	err = copier.CopyWithOption(&reply.Comment, &comment, copier.Option{DeepCopy: true})
	if err != nil {
		reply.StatusCode = -1
		reply.StatusMsg = err.Error()
		return reply, nil
	}
	return reply, nil
}
