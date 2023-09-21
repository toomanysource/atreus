package data

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
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
	GetComment(ctx context.Context, commentId uint32) (*Comment, error)
	InsertComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, commentId uint32) error
	GetComments(ctx context.Context, videoId uint32) ([]*Comment, error)
}

type CacheStore interface {
	HasVideo(ctx context.Context, videoId uint32) (bool, error)
	HasComment(ctx context.Context, videoId, commentId uint32) (bool, error)
	SetComment(ctx context.Context, videoId, commentId uint32, value string) error
	DelComment(ctx context.Context, videoId, commentId uint32) error
	GetComments(ctx context.Context, videoId uint32) ([]string, error)
	SetComments(ctx context.Context, videoId uint32, value interface{}) error
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
	ctx context.Context, userId, videoId, commentId uint32,
) error {
	c, err := r.db.GetComment(ctx, commentId)
	if err != nil {
		return err
	}
	if c.UserId != userId {
		return ErrUserConflict
	}
	if c.VideoId != videoId {
		return ErrVideoConflict
	}

	err = r.db.DeleteComment(ctx, commentId)
	if err != nil {
		return err
	}
	go func() {
		if err = kafkaX.Update(r.kfk, strconv.Itoa(int(videoId)), "-1"); err != nil {
			r.log.Error(err)
		}
	}()
	go func() {
		ctx = context.Background()
		ok, err := r.cache.HasComment(ctx, videoId, commentId)
		if err != nil {
			r.log.Error(err)
			return
		}
		if ok {
			return
		}
		err = r.cache.DelComment(ctx, videoId, commentId)
		if err != nil {
			r.log.Error(err)
		}
	}()
	return nil
}

// CreateComment 创建评论
func (r *commentRepo) CreateComment(
	ctx context.Context, userId, videoId uint32, commentText string, createTime string,
) (*biz.Comment, error) {
	comment := &Comment{
		UserId:   userId,
		VideoId:  videoId,
		Content:  commentText,
		CreateAt: createTime,
	}
	// 先在数据库中插入关系
	err := r.db.InsertComment(ctx, comment)
	if err != nil {
		return nil, err
	}
	go func() {
		if err = kafkaX.Update(r.kfk, strconv.Itoa(int(videoId)), "1"); err != nil {
			r.log.Error(err)
		}
	}()

	go func() {
		ctx = context.Background()
		ok, err := r.cache.HasVideo(ctx, videoId)
		if err != nil {
			r.log.Error(err)
			return
		}
		if !ok {
			return
		}
		marc, err := json.Marshal(comment)
		if err != nil {
			r.log.Error(err)
			return
		}
		if err = r.cache.SetComment(ctx, videoId, comment.Id, string(marc)); err != nil {
			r.log.Error(err)
		}
	}()
	c := new(biz.Comment)
	copier.Copy(c, comment)
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
		l, err := r.cache.GetComments(ctx, videoId)
		if err != nil {
			return nil, err
		}
		for _, v := range l {
			if v == OccupyValue {
				continue
			}
			co := &Comment{}
			if err = json.Unmarshal([]byte(v), co); err != nil {
				return nil, errors.Join(ErrJsonMarshal, err)
			}
			cl = append(cl, co)
		}
		sortComments(cl)
	} else {
		cl, err = r.db.GetComments(ctx, videoId)
		if err != nil {
			return nil, err
		}
		go func(l []*Comment) {
			insertMap := make(map[string]interface{}, len(l))
			// 设置占位键值，防止缓存穿透
			insertMap[OccupyKey] = OccupyValue
			for _, v := range cl {
				marc, err := json.Marshal(v)
				if err != nil {
					r.log.Error(errors.Join(ErrJsonMarshal, err))
					return
				}
				insertMap[strconv.Itoa(int(v.Id))] = marc
			}

			if err = r.cache.SetComments(context.Background(), videoId, insertMap); err != nil {
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

// sortComments 对评论列表进行排序
func sortComments(cl []*Comment) {
	// 对原始切片进行排序
	sort.Slice(cl, func(i, j int) bool {
		return cl[i].Id > cl[j].Id
	})
}
