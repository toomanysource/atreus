package datastore

import (
	"context"
	"errors"

	"github.com/toomanysource/atreus/app/comment/service/internal/data"

	"gorm.io/gorm"

	"github.com/toomanysource/atreus/middleware"
	"github.com/toomanysource/atreus/pkg/errorX"
)

type dbStore struct {
	db *gorm.DB
}

func NewDBStore(db *gorm.DB) data.DBStore {
	return &dbStore{
		db: db,
	}
}

// DeleteComment 数据库删除评论
func (r *dbStore) DeleteComment(
	ctx context.Context, videoId, commentId uint32,
) error {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND video_id = ?", commentId, userId, videoId).
		Delete(&data.Comment{})
	if result.Error != nil {
		return errors.Join(errorX.ErrMysqlDelete, result.Error)
	}
	if result.RowsAffected == 0 {
		return data.ErrProvideInfo
	}
	return nil
}

// InsertComment 数据库插入评论
func (r *dbStore) InsertComment(
	ctx context.Context, videoId uint32, commentText, createTime string,
) (*data.Comment, error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	comment := &data.Comment{
		UserId:   userId,
		VideoId:  videoId,
		Content:  commentText,
		CreateAt: createTime,
	}
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return nil, errors.Join(errorX.ErrMysqlInsert, err)
	}
	return comment, nil
}

// GetComments 数据库搜索评论列表
func (r *dbStore) GetComments(ctx context.Context, videoId uint32) (c []*data.Comment, err error) {
	if err = r.db.WithContext(ctx).Where("video_id = ?", videoId).Find(&c).Error; err != nil {
		return nil, errors.Join(errorX.ErrMysqlQuery, err)
	}
	return c, nil
}
