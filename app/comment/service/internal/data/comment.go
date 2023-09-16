package data

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/toomanysource/atreus/pkg/kafkaX"

	"github.com/segmentio/kafka-go"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/middleware"

	"github.com/toomanysource/atreus/app/comment/service/internal/server"

	"github.com/jinzhu/copier"

	"github.com/toomanysource/atreus/app/comment/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	OccupyKey   = "-1"
	OccupyValue = ""
)

type Comment struct {
	Id       uint32 `gorm:"primary_key"`
	UserId   uint32 `gorm:"column:user_id;not null"`
	VideoId  uint32 `gorm:"column:video_id;not null;index:idx_video_id"`
	Content  string `gorm:"column:content;not null"`
	CreateAt string `gorm:"column:created_at;default:''" copier:"CreateDate"`
}

func (Comment) TableName() string {
	return "comments"
}

type UserRepo interface {
	GetUserInfos(context.Context, uint32, []uint32) ([]*biz.User, error)
}

type commentRepo struct {
	data     *Data
	kfk      *kafka.Writer
	userRepo UserRepo
	log      *log.Helper
}

func NewCommentRepo(
	data *Data, userConn server.UserConn, logger log.Logger,
) biz.CommentRepo {
	return &commentRepo{
		data:     data,
		kfk:      data.kfk,
		userRepo: NewUserRepo(userConn),
		log:      log.NewHelper(log.With(logger, "model", "data/comment")),
	}
}

// DeleteComment 删除评论
func (r *commentRepo) DeleteComment(
	ctx context.Context, videoId, commentId uint32,
) (*biz.Comment, error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	// 先在数据库中删除关系
	err := r.DeleteCommentById(ctx, videoId, commentId)
	if err != nil {
		return nil, err
	}

	go func() {
		if err = r.DeleteCache(context.Background(), videoId, commentId); err != nil {
			r.log.Error(err)
			return
		}
	}()

	r.log.Infof(
		"DeleteComment -> videoId: %v - userId: %v - commentId: %v", videoId, userId, commentId)
	return nil, nil
}

// CreateComment 创建评论
func (r *commentRepo) CreateComment(
	ctx context.Context, videoId uint32, commentText string,
) (*biz.Comment, error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	// 先在数据库中插入关系
	co, err := r.InsertComment(ctx, videoId, commentText, userId)
	if err != nil {
		return nil, err
	}

	go func() {
		if err = r.InsertCache(context.Background(), videoId, co); err != nil {
			r.log.Error(err)
			return
		}
		r.log.Info("redis store success")
	}()

	users, err := r.userRepo.GetUserInfos(ctx, userId, []uint32{userId})
	if err != nil {
		return nil, err
	}
	user := new(biz.User)
	err = copier.Copy(user, users[0])
	if err != nil {
		return nil, errors.Join(ErrCopy, err)
	}
	user.IsFollow = false
	c := new(biz.Comment)
	if err = copier.Copy(c, co); err != nil {
		return nil, errors.Join(ErrCopy, err)
	}
	c.User = user

	r.log.Infof(
		"CreateComment -> videoId: %v - userId: %v - comment: %v", videoId, userId, commentText)
	return c, nil
}

// GetComments 获取评论列表
func (r *commentRepo) GetComments(
	ctx context.Context, videoId uint32,
) (cls []*biz.Comment, err error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	// 先在redis缓存中查询是否存在视频评论列表
	ok, err := r.CheckCache(ctx, videoId)
	if err != nil {
		return nil, err
	}
	var cl []*Comment
	if ok {
		// 如果存在则直接返回
		cl, err = r.GetCache(ctx, videoId)
		if err != nil {
			return nil, err
		}
	} else {
		cl, err = r.GetCommentsByVideoId(ctx, videoId)
		if err != nil {
			return nil, err
		}

		go func(l []*Comment) {
			if err = r.CreateCacheByTrans(context.Background(), l, videoId); err != nil {
				r.log.Error(err)
				return
			}
			r.log.Info("redis transaction success")
		}(cl)
	}

	if len(cl) == 0 {
		return nil, nil
	}
	// 获取评论列表中的所有用户id
	userIds := make([]uint32, 0, len(cl))
	for _, comment := range cl {
		userIds = append(userIds, comment.UserId)
	}

	// 统一查询，减少网络IO
	users, err := r.userRepo.GetUserInfos(ctx, userId, userIds)
	if err != nil {
		return nil, err
	}
	cls = make([]*biz.Comment, 0, len(cl))
	for i, comment := range cl {
		cls = append(cls, &biz.Comment{
			Id:         comment.Id,
			User:       users[i],
			Content:    comment.Content,
			CreateDate: comment.CreateAt,
		})
	}
	sortComments(cls)
	r.log.Infof(
		"GetCommentList -> videoId: %v - commentList: %v", videoId, cls)
	return
}

// InsertCache 创建缓存
func (r *commentRepo) InsertCache(ctx context.Context, videoId uint32, co *Comment) error {
	// 在redis缓存中查询是否存在视频评论列表
	ok, err := r.CheckCache(ctx, videoId)
	if err != nil {
		return err
	}
	if ok {
		// 将评论存入redis缓存
		marc, err := json.Marshal(co)
		if err != nil {
			return errors.Join(ErrJsonMarshal, err)
		}
		if err = r.data.cache.HSet(
			ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(co.Id)), marc).Err(); err != nil {
			return errors.Join(ErrRedisSet, err)
		}
	}
	return nil
}

// DeleteCache 删除缓存
func (r *commentRepo) DeleteCache(ctx context.Context, videoId, commentId uint32) error {
	// 在redis缓存中查询是否存在
	ok, err := r.data.cache.HExists(ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId))).Result()
	if err != nil {
		return errors.Join(ErrRedisQuery, err)
	}
	if ok {
		// 如果存在则删除
		if err = r.data.cache.HDel(ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId))).Err(); err != nil {
			return errors.Join(ErrRedisDelete, err)
		}
	}
	return nil
}

// GetCache 获取缓存
func (r *commentRepo) GetCache(ctx context.Context, videoId uint32) (cl []*Comment, err error) {
	comments, err := r.data.cache.HVals(ctx, strconv.Itoa(int(videoId))).Result()
	if err != nil {
		return nil, errors.Join(ErrRedisQuery, err)
	}

	for _, v := range comments {
		if v == OccupyValue {
			continue
		}
		co := &Comment{}
		if err = json.Unmarshal([]byte(v), co); err != nil {
			return nil, errors.Join(ErrJsonMarshal, err)
		}
		cl = append(cl, co)
	}
	return cl, nil
}

// CheckCache 检查缓存
func (r *commentRepo) CheckCache(ctx context.Context, videoId uint32) (bool, error) {
	// 先在redis缓存中查询是否存在视频评论列表
	count, err := r.data.cache.Exists(ctx, strconv.Itoa(int(videoId))).Result()
	if err != nil {
		return false, errors.Join(ErrRedisQuery, err)
	}
	return count > 0, nil
}

// DeleteCommentById 数据库删除评论
func (r *commentRepo) DeleteCommentById(
	ctx context.Context, videoId, commentId uint32,
) error {
	result := r.data.db.WithContext(ctx).Select("id").Delete(&Comment{}, commentId)
	if result.Error != nil {
		return errors.Join(ErrMysqlDelete, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrInvalidComment
	}
	go func() {
		if err := kafkaX.Update(r.kfk, strconv.Itoa(int(videoId)), "-1"); err != nil {
			r.log.Error(err)
		}
	}()
	return nil
}

// InsertComment 数据库插入评论
func (r *commentRepo) InsertComment(
	ctx context.Context, videoId uint32, commentText string, userId uint32,
) (*Comment, error) {
	comment := &Comment{
		UserId:   userId,
		VideoId:  videoId,
		Content:  commentText,
		CreateAt: time.Now().Format("01-02"),
	}
	if err := r.data.db.WithContext(ctx).Create(comment).Error; err != nil {
		return nil, errors.Join(ErrMysqlInsert, err)
	}
	go func() {
		strconv.Itoa(int(videoId))
		if err := kafkaX.Update(r.kfk, strconv.Itoa(int(videoId)), "1"); err != nil {
			r.log.Error(err)
		}
	}()
	return comment, nil
}

// GetCommentsByVideoId 数据库搜索评论列表
func (r *commentRepo) GetCommentsByVideoId(ctx context.Context, videoId uint32) (c []*Comment, err error) {
	if err = r.data.db.WithContext(ctx).Where("video_id = ?", videoId).Find(&c).Error; err != nil {
		return nil, errors.Join(ErrMysqlQuery, err)
	}
	return c, nil
}

// CreateCacheByTrans 使用事务创建缓存
func (r *commentRepo) CreateCacheByTrans(ctx context.Context, cl []*Comment, videoId uint32) error {
	// 使用事务将评论列表存入redis缓存
	_, err := r.data.cache.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		insertMap := make(map[string]interface{}, len(cl))
		insertMap[OccupyKey] = OccupyValue
		for _, v := range cl {
			marc, err := json.Marshal(v)
			if err != nil {
				return errors.Join(ErrJsonMarshal, err)
			}
			insertMap[strconv.Itoa(int(v.Id))] = marc
		}
		err := pipe.HMSet(ctx, strconv.Itoa(int(videoId)), insertMap).Err()
		if err != nil {
			return errors.Join(ErrRedisSet, err)
		}
		// 将评论数量存入redis缓存,使用随机过期时间防止缓存雪崩
		// 随机生成时间范围
		begin, end := 360, 720
		err = pipe.Expire(ctx, strconv.Itoa(int(videoId)), randomTime(time.Minute, begin, end)).Err()
		if err != nil {
			return errors.Join(ErrRedisSet, err)
		}
		return nil
	})
	if err != nil {
		return errors.Join(ErrRedisTransaction, err)
	}
	return nil
}

// randomTime 随机生成时间
func randomTime(timeType time.Duration, begin, end int) time.Duration {
	return timeType * time.Duration(rand.Intn(end-begin+1)+begin)
}

// sortComments 对评论列表进行排序
func sortComments(cl []*biz.Comment) {
	// 对原始切片进行排序
	sort.Slice(cl, func(i, j int) bool {
		return cl[i].Id > cl[j].Id
	})
}
