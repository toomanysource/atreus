package data

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/toomanysource/atreus/pkg/kafkaX"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/app/favorite/service/internal/biz"
	"github.com/toomanysource/atreus/app/favorite/service/internal/server"

	"github.com/go-kratos/kratos/v2/log"
)

type Favorite struct {
	ID      uint32 `gorm:"column:id;primary_key;autoIncrement"`
	UserID  uint32 `gorm:"column:user_id;index:idx_user_video"`
	VideoID uint32 `gorm:"column:video_id;index:idx_user_video"`
}

func (Favorite) TableName() string {
	return "favorites"
}

type favoriteRepo struct {
	data        *Data
	kfk         KfkWriter
	publishRepo biz.PublishRepo
	log         *log.Helper
}

func NewFavoriteRepo(
	data *Data, publishConn server.PublishConn, logger log.Logger,
) biz.FavoriteRepo {
	return &favoriteRepo{
		data:        data,
		publishRepo: NewPublishRepo(publishConn),
		kfk:         data.kfk,
		log:         log.NewHelper(log.With(logger, "module", "favorite-service/repo")),
	}
}

func (r *favoriteRepo) CreateFavorite(ctx context.Context, userId, videoId uint32) error {
	// 先在数据库中插入关系
	err := r.InsertFavorite(ctx, userId, videoId)
	if err != nil {
		return err
	}
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		count, err := r.data.cache.Exists(ctx, strconv.Itoa(int(userId))).Result()
		if err != nil {
			r.log.Errorf("redis query error %w", err)
			return
		}
		if count == 0 {
			// 如果不存在则创建
			cl, err := r.GetFavorites(ctx, userId)
			if err != nil {
				r.log.Errorf("mysql query error %w", err)
				return
			}
			if err = CacheCreateFavoriteTransaction(ctx, r.data.cache, cl, userId); err != nil {
				r.log.Errorf("redis transaction error %w", err)
				return
			}
			r.log.Info("redis transaction success")
			return
		}
		// 如果存在则更新
		if err = r.data.cache.HSet(
			ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(videoId)), "").Err(); err != nil {
			r.log.Errorf("redis store error %w", err)
			return
		}
		r.log.Info("redis store success")
	}()
	r.log.Infof(
		"CreateFavorite -> userId: %v - videoId: %v", userId, videoId)
	return nil
}

func (r *favoriteRepo) DeleteFavorite(ctx context.Context, userId, videoId uint32) error {
	err := r.DelFavorite(ctx, userId, videoId)
	if err != nil {
		return err
	}
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.data.cache.HExists(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(videoId))).Result()
		if err != nil {
			r.log.Errorf("redis query error %w", err)
			return
		}
		if ok {
			// 如果存在则删除
			if err = r.data.cache.HDel(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(videoId))).Err(); err != nil {
				r.log.Errorf("redis delete error %w", err)
				return
			}
		}
	}()
	r.log.Infof(
		"DeleteFavorite -> userId: %v - videoId: %v", userId, videoId)
	return nil
}

func (r *favoriteRepo) GetFavoriteList(ctx context.Context, userID uint32) ([]biz.Video, error) {
	// 先在redis缓存中查询是否存在喜爱列表
	count, err := r.data.cache.Exists(ctx, strconv.Itoa(int(userID))).Result()
	if err != nil {
		return nil, fmt.Errorf("redis query error %w", err)
	}
	var fl []uint32
	if count > 0 {
		favorites, err := r.data.cache.HKeys(ctx, strconv.Itoa(int(userID))).Result()
		if err != nil {
			return nil, fmt.Errorf("redis query error %w", err)
		}
		for _, v := range favorites {
			if v == "-1" {
				continue
			}
			vc, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("strconv error %w", err)
			}
			fl = append(fl, uint32(vc))
		}
	} else {
		// 如果不存在则创建
		fl, err = r.GetFavorites(ctx, userID)
		if err != nil {
			return nil, err
		}
		// 将喜爱列表存入redis缓存
		go func(l []uint32) {
			if err = CacheCreateFavoriteTransaction(context.Background(), r.data.cache, l, userID); err != nil {
				r.log.Errorf("redis transaction error %w", err)
				return
			}
			r.log.Info("redis transaction success")
		}(fl)
	}
	if len(fl) == 0 {
		return nil, nil
	}
	videos, err := r.publishRepo.GetVideoListByVideoIds(ctx, userID, fl)
	if err != nil {
		return nil, fmt.Errorf("failed to get video info by video ids: %w", err)
	}
	for _, video := range videos {
		video.IsFavorite = true
	}
	r.log.Infof(
		"GetFavoriteVideoList -> userId: %v - videoIdList: %v", userID, fl)
	return videos, nil
}

func (r *favoriteRepo) IsFavorite(ctx context.Context, userId uint32, videoIds []uint32) (oks []bool, err error) {
	count, err := r.data.cache.Exists(ctx, strconv.Itoa(int(userId))).Result()
	if err != nil {
		return nil, fmt.Errorf("redis query error %w", err)
	}
	if count > 0 {
		once := make(map[uint32]bool)
		for _, v := range videoIds {
			if _, ok := once[v]; ok {
				oks = append(oks, once[v])
				continue
			}
			ok, err := r.data.cache.HExists(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(v))).Result()
			if err != nil {
				return nil, fmt.Errorf("redis query error %w", err)
			}
			once[v] = ok
			oks = append(oks, ok)
		}
		return oks, nil
	}
	go func() {
		// 如果不存在则创建
		fl, err := r.GetFavorites(ctx, userId)
		if err != nil {
			r.log.Errorf("mysql query error %w", err)
			return
		}
		// 将喜爱列表存入redis缓存
		if err = CacheCreateFavoriteTransaction(context.Background(), r.data.cache, fl, userId); err != nil {
			r.log.Errorf("redis transaction error %w", err)
			return
		}
		r.log.Info("redis transaction success")
	}()
	return r.CheckFavorite(ctx, userId, videoIds)
}

func (r *favoriteRepo) InsertFavorite(ctx context.Context, userId, videoId uint32) error {
	authorId, err := r.GetAuthorId(ctx, userId, videoId)
	if err != nil {
		return fmt.Errorf("failed to fetch video author: %w", err)
	}
	if err = r.data.db.WithContext(ctx).Create(&Favorite{
		VideoID: videoId,
		UserID:  userId,
	}).Error; err != nil {
		return err
	}
	go func() {
		if err = kafkaX.Update(r.kfk.Favored, authorId, 1); err != nil {
			r.log.Errorf("updateFavorited err: %w", err)
		}
	}()
	go func() {
		if err = kafkaX.Update(r.kfk.Favorite, userId, 1); err != nil {
			r.log.Errorf("updateFavorite err: %w", err)
		}
	}()
	go func() {
		if err = kafkaX.Update(r.kfk.videoFavorite, videoId, 1); err != nil {
			r.log.Errorf("updateFavoriteCount err: %w", err)
		}
	}()
	return nil
}

func (r *favoriteRepo) DelFavorite(ctx context.Context, userId, videoId uint32) error {
	authorId, err := r.GetAuthorId(ctx, userId, videoId)
	if err != nil {
		return errors.New("failed to fetch video author")
	}
	err = r.data.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userId, videoId).Delete(&Favorite{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete favorite: %w", err)
	}
	go func() {
		if err = kafkaX.Update(r.kfk.Favored, authorId, -1); err != nil {
			r.log.Errorf("failed to update favored: %w", err)
		}
	}()
	go func() {
		if err = kafkaX.Update(r.kfk.Favorite, userId, -1); err != nil {
			r.log.Errorf("failed to update favorite: %w", err)
		}
	}()
	go func() {
		if err = kafkaX.Update(r.kfk.videoFavorite, videoId, -1); err != nil {
			r.log.Errorf("failed to update favorite count: %w", err)
		}
	}()
	return nil
}

func (r *favoriteRepo) GetFavorites(ctx context.Context, userID uint32) ([]uint32, error) {
	var favorites []Favorite
	result := r.data.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&favorites)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get favorite list: %w", result.Error)
	}
	if len(favorites) == 0 {
		return nil, nil
	}

	videoIDs := make([]uint32, 0, len(favorites))
	for _, favorite := range favorites {
		videoIDs = append(videoIDs, favorite.VideoID)
	}
	return videoIDs, nil
}

func (r *favoriteRepo) CheckFavorite(ctx context.Context, userId uint32, videoIds []uint32) ([]bool, error) {
	var favorites []Favorite
	result := r.data.db.WithContext(ctx).
		Where("user_id = ? AND video_id IN ?", userId, videoIds).
		Find(&favorites)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch favorites: %w", result.Error)
	}
	favoriteMap := make(map[uint32]bool, len(favorites))
	for _, favorite := range favorites {
		favoriteMap[favorite.VideoID] = true
	}

	isFavorite := make([]bool, 0, len(videoIds))
	for _, videoId := range videoIds {
		if _, ok := favoriteMap[videoId]; !ok {
			isFavorite = append(isFavorite, false)
			continue
		}
		isFavorite = append(isFavorite, true)
	}
	return isFavorite, nil
}

func (r *favoriteRepo) GetAuthorId(ctx context.Context, userId uint32, videoId uint32) (uint32, error) {
	videoList, err := r.publishRepo.GetVideoListByVideoIds(ctx, userId, []uint32{videoId})
	if err != nil {
		return 0, fmt.Errorf("failed to get video info by video ids: %w", err)
	}
	if len(videoList) == 0 {
		return 0, errors.New("video not found")
	}
	authorId := videoList[0].Author.Id
	return authorId, nil
}

// CacheCreateFavoriteTransaction 缓存创建事务
func CacheCreateFavoriteTransaction(ctx context.Context, cache *redis.Client, vl []uint32, userId uint32) error {
	// 使用事务将列表存入redis缓存
	_, err := cache.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		insertMap := make(map[string]interface{}, len(vl))
		// 如果喜爱列表为空则插入一个空值,防止缓存击穿
		insertMap["-1"] = ""
		for _, v := range vl {
			vs := strconv.Itoa(int(v))
			insertMap[vs] = ""
		}
		err := pipe.HMSet(ctx, strconv.Itoa(int(userId)), insertMap).Err()
		if err != nil {
			return fmt.Errorf("redis store error, err : %w", err)
		}
		// 将评论数量存入redis缓存,使用随机过期时间防止缓存雪崩
		begin, end := 360, 720
		err = pipe.Expire(ctx, strconv.Itoa(int(userId)), randomTime(time.Minute, begin, end)).Err()
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
