package datastore

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"github.com/toomanysource/atreus/app/comment/service/internal/data"

	"github.com/go-redis/redis/v8"
)

const (
	ExpireBegin = 360
	ExpireEnd   = 720
)

type cacheStore struct {
	cache *redis.Client
}

func NewCacheStore(cache *redis.Client) data.CacheStore {
	return &cacheStore{
		cache: cache,
	}
}

// SetComments 使用事务创建缓存
func (r *cacheStore) SetComments(ctx context.Context, videoId uint32, value interface{}) error {
	if err := r.setComments(ctx, videoId, value); err != nil {
		return err
	}
	return r.setExpire(ctx, videoId)
}

// HasVideo 是否存在Video缓存
func (r *cacheStore) HasVideo(ctx context.Context, videoId uint32) (bool, error) {
	// 先在redis缓存中查询是否存在视频评论列表
	count, err := r.cache.Exists(ctx, strconv.Itoa(int(videoId))).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasComment 是否存在Comment缓存
func (r *cacheStore) HasComment(ctx context.Context, videoId, commentId uint32) (bool, error) {
	return r.cache.HExists(ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId))).Result()
}

// GetComments 获取缓存评论列表
func (r *cacheStore) GetComments(ctx context.Context, videoId uint32) (cl []string, err error) {
	return r.cache.HVals(ctx, strconv.Itoa(int(videoId))).Result()
}

// DelComment 删除缓存评论
func (r *cacheStore) DelComment(ctx context.Context, videoId, commentId uint32) error {
	return r.cache.HDel(ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId))).Err()
}

// SetComment 创建缓存评论
func (r *cacheStore) SetComment(ctx context.Context, videoId, commentId uint32, value string) error {
	return r.cache.HSet(ctx, strconv.Itoa(int(videoId)), strconv.Itoa(int(commentId)), value).Err()
}

// setComments 创建缓存评论列表
func (r *cacheStore) setComments(ctx context.Context, videoId uint32, value interface{}) error {
	return r.cache.HMSet(ctx, strconv.Itoa(int(videoId)), value).Err()
}

// setExpire 设置Key过期时间
func (r *cacheStore) setExpire(ctx context.Context, videoId uint32) error {
	// 将评论数量存入redis缓存,使用随机过期时间防止缓存雪崩
	return r.cache.Expire(
		ctx, strconv.Itoa(int(videoId)), randomTime(time.Minute, ExpireBegin, ExpireEnd)).Err()
}

// randomTime 随机生成时间
func randomTime(timeType time.Duration, begin, end int) time.Duration {
	return timeType * time.Duration(rand.Intn(end-begin+1)+begin)
}
