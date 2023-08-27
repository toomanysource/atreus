package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/middleware"

	"github.com/toomanysource/atreus/app/comment/service/internal/server"

	"github.com/jinzhu/copier"
	"gorm.io/gorm"

	"github.com/toomanysource/atreus/app/comment/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// Comment Database Model
type Comment struct {
	Id       uint32 `gorm:"primary_key"`
	UserId   uint32 `gorm:"column:user_id;not null;index"`
	VideoId  uint32 `gorm:"column:video_id;not null;index"`
	Content  string `gorm:"column:content;not null"`
	CreateAt string `gorm:"column:created_at;default:''"`
}

func (Comment) TableName() string {
	return "comments"
}

type PublishRepo interface {
	UpdateComment(context.Context, uint32, int32) error
}

type UserRepo interface {
	GetUserInfos(context.Context, uint32, []uint32) ([]*biz.User, error)
}

type commentRepo struct {
	data        *Data
	publishRepo PublishRepo
	userRepo    UserRepo
	log         *log.Helper
}

func NewCommentRepo(
	data *Data, userConn server.UserConn, publishConn server.PublishConn, logger log.Logger,
) biz.CommentRepo {
	return &commentRepo{
		data:        data,
		publishRepo: NewPublishRepo(publishConn),
		userRepo:    NewUserRepo(userConn),
		log:         log.NewHelper(log.With(logger, "model", "comment/repo")),
	}
}

// DeleteComment 删除评论，先在数据库中删除，再在redis缓存中删除
func (r *commentRepo) DeleteComment(
	ctx context.Context, videoId, commentId uint32,
) (*biz.Comment, error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	// 先在数据库中删除关系
	err := r.DelComment(ctx, videoId, commentId, userId)
	if err != nil {
		return nil, err
	}
	go func() {
		// 设置goroutine内部上下文填空，防止外部上下文取消导致goroutine退出
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.data.cache.HExists(ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId))).Result()
		if err != nil {
			r.log.Errorf("redis query error %w", err)
			return
		}
		if ok {
			// 如果存在则删除
			if err = r.data.cache.HDel(ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId))).Err(); err != nil {
				r.log.Errorf("redis delete error %w", err)
				return
			}
		}
	}()
	r.log.Infof(
		"DeleteComment -> videoId: %v - userId: %v - commentId: %v", videoId, userId, commentId)
	return nil, nil
}

// CreateComment 创建评论
func (r *commentRepo) CreateComment(
	ctx context.Context, videoId uint32, commentText string,
) (c *biz.Comment, err error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	// 先在数据库中插入关系
	co, err := r.InsertComment(ctx, videoId, commentText, userId)
	if err != nil {
		return nil, err
	}
	go func() {
		ctx = context.TODO()
		// 在redis缓存中查询是否存在视频评论列表
		count, err := r.data.cache.Exists(ctx, strconv.Itoa(int(videoId))).Result()
		if err != nil {
			r.log.Errorf("redis query error %w", err)
			return
		}
		if count == 0 {
			// 如果不存在则创建
			cl, err := r.SearchCommentList(ctx, videoId)
			if err != nil {
				r.log.Errorf("mysql query error %w", err)
				return
			}
			if err = r.CacheCreateCommentTransaction(ctx, cl, videoId); err != nil {
				r.log.Errorf("redis transaction error %w", err)
				return
			}
			r.log.Info("redis transaction success")
			return
		}
		// 将评论存入redis缓存
		marc, err := json.Marshal(co)
		if err = r.data.cache.HSet(
			ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(co.Id)), marc).Err(); err != nil {
			r.log.Errorf("redis store error %w", err)
			return
		}
		r.log.Info("redis store success")
	}()
	users, err := r.userRepo.GetUserInfos(ctx, 0, []uint32{userId})
	if err != nil {
		return nil, fmt.Errorf("user service transfer error %w", err)
	}
	var user biz.User
	err = copier.Copy(&user, &users[0])
	if err != nil {
		return nil, fmt.Errorf("data replication error, err : %w", err)
	}
	user.IsFollow = false
	c = &biz.Comment{
		Id:         co.Id,
		User:       &user,
		Content:    co.Content,
		CreateDate: co.CreateAt,
	}
	r.log.Infof(
		"CreateComment -> videoId: %v - userId: %v - comment: %v", videoId, userId, commentText)
	return
}

// GetCommentList 获取评论列表
func (r *commentRepo) GetCommentList(
	ctx context.Context, videoId uint32,
) (cls []*biz.Comment, err error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	if videoId == 0 {
		return nil, errors.New("videoId is empty")
	}
	// 先在redis缓存中查询是否存在视频评论列表
	comments, err := r.data.cache.HKeys(ctx, strconv.Itoa(int(videoId))).Result()
	if err != nil {
		return nil, fmt.Errorf("redis query error %w", err)
	}

	cl := make([]*Comment, len(comments))
	if len(comments) > 0 {
		// 如果存在则直接返回
		var wg sync.WaitGroup
		var mutex sync.Mutex
		errChan := make(chan error)
		for _, v := range comments {
			wg.Add(1)
			go func(comment string) {
				defer wg.Done()
				co := &Comment{}
				if err = json.Unmarshal([]byte(comment), co); err != nil {
					errChan <- fmt.Errorf("json unmarshal error %w", err)
					return
				}
				mutex.Lock()
				cl = append(cl, co)
				mutex.Unlock()
			}(v)
		}
		wg.Wait()
		if err = <-errChan; err != nil {
			return nil, err
		}
	} else {
		cl, err = r.SearchCommentList(ctx, videoId)
		if err != nil {
			return nil, err
		}
		// 没有列表则不创建
		if len(cl) == 0 {
			return nil, nil
		}
		go func(l []*Comment) {
			if err = r.CacheCreateCommentTransaction(context.Background(), l, videoId); err != nil {
				r.log.Errorf("redis transaction error %w", err)
				return
			}
			r.log.Info("redis transaction success")
		}(cl)
	}
	// 获取评论列表中的所有用户id
	userIds := make([]uint32, 0, len(cl))
	for _, comment := range cl {
		userIds = append(userIds, comment.UserId)
	}

	// 统一查询，减少网络IO
	users, err := r.userRepo.GetUserInfos(ctx, userId, userIds)
	if err != nil {
		return nil, fmt.Errorf("user search data error %w", err)
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

// DelComment 数据库删除评论
func (r *commentRepo) DelComment(
	ctx context.Context, videoId, commentId uint32, userId uint32,
) error {
	comment := &Comment{}
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.First(comment, commentId)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if result.Error != nil {
			return fmt.Errorf("mysql query error %w", result.Error)
		}
		// 判断当前用户是否为评论用户
		if comment.UserId != userId {
			return errors.New("comment user conflict")
		}
		// 判断视频id是否为当前视频id
		if comment.VideoId != videoId {
			return errors.New("comment video conflict")
		}
		if err := tx.Select("id").Delete(&Comment{}, commentId).Error; err != nil {
			return fmt.Errorf("mysql delete error %w", err)
		}
		if err := r.publishRepo.UpdateComment(ctx, videoId, -1); err != nil {
			return fmt.Errorf("publish update data error %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("mysql transaction error %w", err)
	}
	return nil
}

// InsertComment 数据库插入评论
func (r *commentRepo) InsertComment(
	ctx context.Context, videoId uint32, commentText string, userId uint32,
) (*Comment, error) {
	if commentText == "" {
		return nil, errors.New("comment text not exist")
	}
	comment := &Comment{
		UserId:   userId,
		VideoId:  videoId,
		Content:  commentText,
		CreateAt: time.Now().Format("01-02"),
	}
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return fmt.Errorf("mysql create error %w", err)
		}
		if err := r.publishRepo.UpdateComment(ctx, videoId, 1); err != nil {
			return fmt.Errorf("publish update data error %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("mysql transaction error %w", err)
	}
	return comment, nil
}

// SearchCommentList 数据库搜索评论列表
func (r *commentRepo) SearchCommentList(ctx context.Context, videoId uint32) ([]*Comment, error) {
	var commentList []*Comment
	result := r.data.db.WithContext(ctx).Where("video_id = ?", videoId).Find(&commentList)
	if err := result.Error; err != nil {
		return nil, fmt.Errorf("mysql query error %w", err)
	}
	// 此视频没有评论
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return commentList, nil
}

// CacheCreateCommentTransaction 缓存创建事务
func (r *commentRepo) CacheCreateCommentTransaction(ctx context.Context, cl []*Comment, videoId uint32) error {
	// 使用事务将评论列表存入redis缓存
	_, err := r.data.cache.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		insertMap := make(map[string]interface{}, len(cl))
		for _, v := range cl {
			marc, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("json marshal error, err : %w", err)
			}
			insertMap[strconv.Itoa(int(v.Id))] = marc
		}
		err := pipe.HMSet(ctx, strconv.Itoa(int(videoId)), insertMap).Err()
		if err != nil {
			return fmt.Errorf("redis store error, err : %w", err)
		}
		// 将评论数量存入redis缓存,使用随机过期时间防止缓存雪崩
		// 随机生成时间范围
		begin, end := 360, 720
		err = pipe.Expire(ctx, strconv.Itoa(int(videoId)), randomTime(time.Minute, begin, end)).Err()
		if err != nil {
			return fmt.Errorf("redis expire error, err : %w", err)
		}
		return nil
	})
	return err
}

// randomTime 随机生成时间
func randomTime(timeType time.Duration, begin, end int) time.Duration {
	return timeType * time.Duration(rand.Intn(end-begin+1)+begin)
}

// sortComments 对评论列表进行排序
func sortComments(cl []*biz.Comment) {
	// 对原始切片进行排序
	sort.Slice(cl, func(i, j int) bool {
		t1, _ := time.Parse("01-02", cl[i].CreateDate)
		t2, _ := time.Parse("01-02", cl[j].CreateDate)
		return t1.After(t2)
	})
}
