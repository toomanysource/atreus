package biz

import (
	"errors"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewCommentUseCase)

const (
	CreateType uint32 = 1
	DeleteType uint32 = 2
)

var (
	ErrCommentTextEmpty  = errors.New("评论内容不能为空")
	ErrInValidActionType = errors.New("错误的行为")
	ErrCommentTextSafety = errors.New("评论内容存在不安全因素")
	ErrCreateComment     = errors.New("创建评论失败")
	ErrDeleteComment     = errors.New("删除评论失败")
	ErrGetCommentList    = errors.New("获取评论列表失败")
)
