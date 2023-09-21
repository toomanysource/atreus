package datastore

import (
	"context"

	"github.com/toomanysource/atreus/app/comment/service/internal/data"

	"gorm.io/gorm"
)

type dbStore struct {
	db *gorm.DB
}

func NewDBStore(db *gorm.DB) data.DBStore {
	return &dbStore{
		db: db,
	}
}

// GetComment 数据库搜索评论
func (r *dbStore) GetComment(ctx context.Context, commentId uint32) (c *data.Comment, err error) {
	err = r.db.WithContext(ctx).First(&c, commentId).Error
	return
}

// DeleteComment 数据库删除评论
func (r *dbStore) DeleteComment(ctx context.Context, commentId uint32) error {
	return r.db.WithContext(ctx).Delete(&data.Comment{}, commentId).Error
}

// InsertComment 数据库插入评论
func (r *dbStore) InsertComment(ctx context.Context, comment *data.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

// GetComments 数据库搜索评论列表
func (r *dbStore) GetComments(ctx context.Context, videoId uint32) (c []*data.Comment, err error) {
	err = r.db.WithContext(ctx).Where("video_id = ?", videoId).Order("id Desc").Find(&c).Error
	return
}
