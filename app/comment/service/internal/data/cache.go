package data

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/pkg/errorX"
)

type cacheRepo struct {
	cache *redis.Client
}

func NewCacheRepo(cache *redis.Client) CacheRepo {
	return &cacheRepo{
		cache: cache,
	}
}

// InsertComment 创建缓存
func (r *cacheRepo) InsertComment(ctx context.Context, videoId uint32, co *Comment) error {
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
func (r *cacheRepo) DeleteComment(ctx context.Context, videoId, commentId uint32) error {
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
func (r *cacheRepo) GetComments(ctx context.Context, videoId uint32) (cl []*Comment, err error) {
	return r.getComments(ctx, videoId)
}

// InsertComments 使用事务创建缓存
func (r *cacheRepo) InsertComments(ctx context.Context, cl []*Comment, videoId uint32) error {
	return r.setComments(ctx, cl, videoId)
}

// HasVideo 是否存在Video缓存
func (r *cacheRepo) HasVideo(ctx context.Context, videoId uint32) (bool, error) {
	return r.hasVideo(ctx, videoId)
}

// HasVideo 是否存在Video缓存
func (r *cacheRepo) hasVideo(ctx context.Context, videoId uint32) (bool, error) {
	// 先在redis缓存中查询是否存在视频评论列表
	count, err := r.cache.Exists(ctx, strconv.Itoa(int(videoId))).Result()
	if err != nil {
		return false, errors.Join(errorX.ErrRedisQuery, err)
	}
	return count > 0, nil
}

// hasComment 是否存在Comment缓存
func (r *cacheRepo) hasComment(ctx context.Context, videoId, commentId uint32) (bool, error) {
	ok, err := r.cache.HExists(
		ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId)),
	).Result()
	if err != nil {
		return false, errors.Join(errorX.ErrRedisQuery, err)
	}
	return ok, nil
}

// getComments 获取缓存评论列表
func (r *cacheRepo) getComments(ctx context.Context, videoId uint32) (cl []*Comment, err error) {
	comments, err := r.cache.HVals(ctx, strconv.Itoa(int(videoId))).Result()
	if err != nil {
		return nil, errors.Join(errorX.ErrRedisQuery, err)
	}

	for _, v := range comments {
		if v == OccupyValue {
			continue
		}
		co := &Comment{}
		if err = json.Unmarshal([]byte(v), co); err != nil {
			return nil, errors.Join(errorX.ErrJsonMarshal, err)
		}
		cl = append(cl, co)
	}
	return cl, nil
}

// delComment 删除缓存评论
func (r *cacheRepo) delComment(ctx context.Context, videoId, commentId uint32) error {
	if err := r.cache.HDel(
		ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId)),
	).Err(); err != nil {
		return errors.Join(errorX.ErrRedisDelete, err)
	}
	return nil
}

// setComment 创建缓存评论
func (r *cacheRepo) setComment(ctx context.Context, videoId uint32, co *Comment) error {
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
func (r *cacheRepo) setComments(ctx context.Context, cl []*Comment, videoId uint32) error {
	// 使用事务将评论列表存入redis缓存
	_, err := r.cache.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		insertMap := make(map[string]interface{}, len(cl))
		insertMap[OccupyKey] = OccupyValue
		for _, v := range cl {
			marc, err := json.Marshal(v)
			if err != nil {
				return errors.Join(errorX.ErrJsonMarshal, err)
			}
			insertMap[strconv.Itoa(int(v.Id))] = marc
		}
		err := pipe.HMSet(ctx, strconv.Itoa(int(videoId)), insertMap).Err()
		if err != nil {
			return errors.Join(errorX.ErrRedisSet, err)
		}
		// 将评论数量存入redis缓存,使用随机过期时间防止缓存雪崩
		// 随机生成时间范围
		begin, end := 360, 720
		err = pipe.Expire(ctx, strconv.Itoa(int(videoId)), randomTime(time.Minute, begin, end)).Err()
		if err != nil {
			return errors.Join(errorX.ErrRedisSet, err)
		}
		return nil
	})
	if err != nil {
		return errors.Join(errorX.ErrRedisTransaction, err)
	}
	return nil
}

// randomTime 随机生成时间
func randomTime(timeType time.Duration, begin, end int) time.Duration {
	return timeType * time.Duration(rand.Intn(end-begin+1)+begin)
}
