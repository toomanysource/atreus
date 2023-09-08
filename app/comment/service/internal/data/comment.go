package data

import (
	"context"
	"strconv"

	"github.com/jinzhu/copier"

	"github.com/toomanysource/atreus/pkg/kafkaX"

	"github.com/segmentio/kafka-go"

	"github.com/toomanysource/atreus/app/comment/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type Comment struct {
	Id       uint32 `gorm:"primary_key;index:idx_id_user_video"`
	UserId   uint32 `gorm:"column:user_id;not null;index:idx_id_user_video"`
	VideoId  uint32 `gorm:"column:video_id;not null;index:idx_id_user_video;index:idx_video_id"`
	Content  string `gorm:"column:content;not null"`
	CreateAt string `gorm:"column:created_at;default:''" copier:"CreateDate"`
}

func (Comment) TableName() string {
	return "comments"
}

type DBStore interface {
	InsertComment(
		ctx context.Context, videoId uint32, commentText, createTime string,
	) (*Comment, error)
	DeleteComment(ctx context.Context, videoId, commentId uint32) error
	GetComments(ctx context.Context, videoId uint32) (c []*Comment, err error)
}

type CacheStore interface {
	InsertComment(ctx context.Context, videoId uint32, co *Comment) error
	DeleteComment(ctx context.Context, videoId, commentId uint32) error
	GetComments(ctx context.Context, videoId uint32) (cl []*Comment, err error)
	InsertComments(ctx context.Context, cl []*Comment, videoId uint32) error
	HasVideo(ctx context.Context, videoId uint32) (bool, error)
}

type commentRepo struct {
	kfk   *kafka.Writer
	db    DBStore
	cache CacheStore
	log   *log.Helper
}

func NewCommentRepo(data *Data, db DBStore, cache CacheStore, logger log.Logger) biz.CommentRepo {
	return &commentRepo{
		kfk:   data.kfk,
		db:    db,
		cache: cache,
		log:   log.NewHelper(log.With(logger, "model", "data/comment")),
	}
}

// DeleteComment 删除评论
func (r *commentRepo) DeleteComment(
	ctx context.Context, videoId, commentId uint32,
) error {
	// 先在数据库中删除关系
	err := r.db.DeleteComment(ctx, videoId, commentId)
	if err != nil {
		return err
	}
	go func() {
		if err = kafkaX.Update(r.kfk, strconv.Itoa(int(videoId)), "-1"); err != nil {
			r.log.Error(err)
		}
	}()
	go func() {
		if err = r.cache.DeleteComment(context.Background(), videoId, commentId); err != nil {
			r.log.Error(err)
		}
	}()
	return nil
}

// CreateComment 创建评论
func (r *commentRepo) CreateComment(
	ctx context.Context, videoId uint32, commentText string, createTime string,
) (*biz.Comment, error) {
	// 先在数据库中插入关系
	co, err := r.db.InsertComment(ctx, videoId, commentText, createTime)
	if err != nil {
		return nil, err
	}
	go func() {
		if err = kafkaX.Update(r.kfk, strconv.Itoa(int(videoId)), "1"); err != nil {
			r.log.Error(err)
		}
	}()

	go func() {
		if err = r.cache.InsertComment(context.Background(), videoId, co); err != nil {
			r.log.Error(err)
		}
	}()
	c := new(biz.Comment)
	copier.Copy(c, co)
	return c, nil
}

// GetComments 获取评论列表
func (r *commentRepo) GetComments(
	ctx context.Context, videoId uint32,
) (cls []*biz.Comment, err error) {
	// 先在redis缓存中查询是否存在视频评论列表
	ok, err := r.cache.HasVideo(ctx, videoId)
	if err != nil {
		return nil, err
	}
	var cl []*Comment
	if ok {
		// 如果存在则直接返回
		cl, err = r.cache.GetComments(ctx, videoId)
		if err != nil {
			return nil, err
		}
	} else {
		cl, err = r.db.GetComments(ctx, videoId)
		if err != nil {
			return nil, err
		}

		go func(l []*Comment) {
			if err = r.cache.InsertComments(context.Background(), l, videoId); err != nil {
				r.log.Error(err)
			}
		}(cl)
	}
	if len(cl) == 0 {
		return nil, nil
	}
	for _, comment := range cl {
		cls = append(cls, &biz.Comment{
			Id: comment.Id,
			User: &biz.User{
				Id: comment.UserId,
			},
			Content:    comment.Content,
			CreateDate: comment.CreateAt,
		})
	}
	return cls, nil
}
