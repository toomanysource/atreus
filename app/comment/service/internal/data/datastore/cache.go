package datastore

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/toomanysource/atreus/app/comment/service/internal/data"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/pkg/errorX"
)

const (
	ExpireBegin = 360
	ExpireEnd   = 720
	OccupyKey   = "-1"
	OccupyValue = ""
)

type cacheStore struct {
	cache *redis.Client
}

func NewCacheStore(cache *redis.Client) data.CacheStore {
	return &cacheStore{
		cache: cache,
	}
}

// InsertComment 创建缓存
func (r *cacheStore) InsertComment(ctx context.Context, videoId uint32, co *data.Comment) error {
	// 在redis缓存中查询是否存在视频评论列表
	ok, err := r.hasVideo(ctx, videoId)
	if err != nil {
		return err
	}
	if ok {
		// 将评论存入redis缓存
		if err = r.setComment(ctx, videoId, co); err != nil {
			return err
		}
	}
	return nil
}

// DeleteComment 删除缓存
func (r *cacheStore) DeleteComment(ctx context.Context, videoId, commentId uint32) error {
	// 在redis缓存中查询是否存在
	ok, err := r.hasComment(ctx, videoId, commentId)
	if err != nil {
		return err
	}
	if !ok {
		// 如果存在则删除
		if err = r.delComment(ctx, videoId, commentId); err != nil {
			return err
		}
	}
	return nil
}

// GetComments 获取缓存
func (r *cacheStore) GetComments(ctx context.Context, videoId uint32) ([]*data.Comment, error) {
	cl, err := r.getComments(ctx, videoId)
	if err != nil {
		return nil, err
	}
	sortComments(cl)
	return cl, nil
}

// InsertComments 使用事务创建缓存
func (r *cacheStore) InsertComments(ctx context.Context, cl []*data.Comment, videoId uint32) error {
	if err := r.setComments(ctx, videoId, cl); err != nil {
		return err
	}
	return r.setExpire(ctx, videoId)
}

// HasVideo 是否存在Video缓存
func (r *cacheStore) HasVideo(ctx context.Context, videoId uint32) (bool, error) {
	return r.hasVideo(ctx, videoId)
}

// HasVideo 是否存在Video缓存
func (r *cacheStore) hasVideo(ctx context.Context, videoId uint32) (bool, error) {
	// 先在redis缓存中查询是否存在视频评论列表
	count, err := r.cache.Exists(ctx, strconv.Itoa(int(videoId))).Result()
	if err != nil {
		return false, errors.Join(errorX.ErrRedisQuery, err)
	}
	return count > 0, nil
}

// hasComment 是否存在Comment缓存
func (r *cacheStore) hasComment(ctx context.Context, videoId, commentId uint32) (bool, error) {
	ok, err := r.cache.HExists(
		ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId)),
	).Result()
	if err != nil {
		return false, errors.Join(errorX.ErrRedisQuery, err)
	}
	return ok, nil
}

// getComments 获取缓存评论列表
func (r *cacheStore) getComments(ctx context.Context, videoId uint32) (cl []*data.Comment, err error) {
	comments, err := r.cache.HVals(ctx, strconv.Itoa(int(videoId))).Result()
	if err != nil {
		return nil, errors.Join(errorX.ErrRedisQuery, err)
	}

	for _, v := range comments {
		if v == OccupyValue {
			continue
		}
		co := &data.Comment{}
		if err = json.Unmarshal([]byte(v), co); err != nil {
			return nil, errors.Join(errorX.ErrJsonMarshal, err)
		}
		cl = append(cl, co)
	}
	return
}

// delComment 删除缓存评论
func (r *cacheStore) delComment(ctx context.Context, videoId, commentId uint32) error {
	if err := r.cache.HDel(
		ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId)),
	).Err(); err != nil {
		return errors.Join(errorX.ErrRedisDelete, err)
	}
	return nil
}

// setComment 创建缓存评论
func (r *cacheStore) setComment(ctx context.Context, videoId uint32, co *data.Comment) error {
	marc, err := json.Marshal(co)
	if err != nil {
		return errors.Join(errorX.ErrJsonMarshal, err)
	}
	if err = r.cache.HSet(
		ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(co.Id)), marc).Err(); err != nil {
		return errors.Join(errorX.ErrRedisSet, err)
	}
	return nil
}

// setComments 创建缓存评论列表
func (r *cacheStore) setComments(ctx context.Context, videoId uint32, cl []*data.Comment) error {
	insertMap := make(map[string]interface{}, len(cl))
	// 设置占位键值，防止缓存穿透
	insertMap[OccupyKey] = OccupyValue
	for _, v := range cl {
		marc, err := json.Marshal(v)
		if err != nil {
			return errors.Join(errorX.ErrJsonMarshal, err)
		}
		insertMap[strconv.Itoa(int(v.Id))] = marc
	}
	err := r.cache.HMSet(ctx, strconv.Itoa(int(videoId)), insertMap).Err()
	if err != nil {
		return errors.Join(errorX.ErrRedisSet, err)
	}
	return nil
}

// setExpire 设置Key过期时间
func (r *cacheStore) setExpire(ctx context.Context, videoId uint32) error {
	// 将评论数量存入redis缓存,使用随机过期时间防止缓存雪崩
	if err := r.cache.Expire(
		ctx, strconv.Itoa(int(videoId)), randomTime(time.Minute, ExpireBegin, ExpireEnd)).
		Err(); err != nil {
		return errors.Join(errorX.ErrRedisSet, err)
	}
	return nil
}

// randomTime 随机生成时间
func randomTime(timeType time.Duration, begin, end int) time.Duration {
	return timeType * time.Duration(rand.Intn(end-begin+1)+begin)
}

// sortComments 对评论列表进行排序
func sortComments(cl []*data.Comment) {
	// 对原始切片进行排序
	sort.Slice(cl, func(i, j int) bool {
		return cl[i].Id > cl[j].Id
	})
}