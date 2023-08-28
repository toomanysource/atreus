package data

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/app/favorite/service/internal/biz"
	"github.com/toomanysource/atreus/app/favorite/service/internal/server"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type Favorite struct {
	ID        uint32    `gorm:"column:id;primary_key;autoIncrement"`
	VideoID   uint32    `gorm:"column:video_id"`
	UserID    uint32    `gorm:"column:user_id"`
	CreatedAt time.Time `gorm:"column:created_at"` // new add field; for backend use only
}

func (Favorite) TableName() string {
	return "favorites"
}

type favoriteRepo struct {
	data        *Data
	publishRepo biz.PublishRepo
	userRepo    biz.UserRepo
	log         *log.Helper
}

func NewFavoriteRepo(
	data *Data, publishConn server.PublishConn, userConn server.UserConn, logger log.Logger,
) biz.FavoriteRepo {
	return &favoriteRepo{
		data:        data,
		publishRepo: NewPublishRepo(publishConn),
		userRepo:    NewUserRepo(userConn),
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
		ok, err := r.data.cache.HExists(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(videoId))).Result()
		if err != nil {
			r.log.Errorf("redis query error %w", err)
			return
		}
		if !ok {
			// 如果不存在则创建
			cl, err := r.GetFavorites(ctx, userId)
			if err != nil {
				r.log.Errorf("mysql query error %w", err)
				return
			}
			// 没有喜爱列表则不创建
			if len(cl) == 0 {
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
	favorites, err := r.data.cache.HKeys(ctx, strconv.Itoa(int(userID))).Result()
	if err != nil {
		return nil, fmt.Errorf("redis query error %w", err)
	}
	fl := make([]uint32, 0, len(favorites))
	if len(favorites) > 0 {
		for _, v := range favorites {
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
		// 没有喜爱列表则不创建
		if len(fl) == 0 {
			return nil, nil
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
		for _, v := range videoIds {
			ok, err := r.data.cache.HExists(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(v))).Result()
			if err != nil {
				return nil, fmt.Errorf("redis query error %w", err)
			}
			oks = append(oks, ok)
		}
		return oks, nil
	}
	return r.CheckFavorite(ctx, userId, videoIds)
}

func (r *favoriteRepo) UpdateVideoFavorite(videoId uint32, change int32) error {
	return r.data.kfk.WriteMessages(context.TODO(), kafka.Message{
		Partition: 0,
		Key:       []byte(strconv.Itoa(int(videoId))),
		Value:     []byte(strconv.Itoa(int(change))),
	})
}

func (r *favoriteRepo) InsertFavorite(ctx context.Context, userId, videoId uint32) error {
	isFavorite, err := r.CheckFavorite(ctx, userId, []uint32{videoId})
	if err != nil {
		return fmt.Errorf("favorite query error: %w", err)
	}
	if isFavorite != nil {
		return nil
	}

	authorId, err := r.GetAuthorId(ctx, userId, videoId)
	if err != nil {
		return fmt.Errorf("failed to fetch video author: %w", err)
	}

	err = r.data.db.Transaction(func(tx *gorm.DB) error {
		if err = tx.WithContext(ctx).Create(&Favorite{
			VideoID: videoId,
			UserID:  userId,
		}).Error; err != nil {
			return err
		}

		if err = r.userRepo.UpdateFavorited(ctx, authorId, 1); err != nil {
			return fmt.Errorf("updateFavorited err: %w", err)
		}
		if err = r.userRepo.UpdateFavorite(ctx, userId, 1); err != nil {
			return fmt.Errorf("updateFavorite err: %w", err)
		}
		if err = r.UpdateVideoFavorite(videoId, 1); err != nil {
			return fmt.Errorf("updateFavoriteCount err: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create favorite: %w", err)
	}
	return nil
}

func (r *favoriteRepo) DelFavorite(ctx context.Context, userId, videoId uint32) error {
	isFavorite, err := r.CheckFavorite(ctx, userId, []uint32{videoId})
	if err != nil {
		return fmt.Errorf("favorite query error: %w", err)
	}
	if isFavorite == nil {
		return nil
	}

	authorId, err := r.GetAuthorId(ctx, userId, videoId)
	if err != nil {
		return errors.New("failed to fetch video author")
	}

	result := r.data.db.Transaction(func(tx *gorm.DB) error {
		err = tx.WithContext(ctx).Where("user_id = ? AND video_id = ?", userId, videoId).Delete(&Favorite{}).Error
		if err != nil {
			return fmt.Errorf("failed to delete favorite: %w", err)
		}

		err = r.userRepo.UpdateFavorited(ctx, authorId, -1)
		if err != nil {
			return fmt.Errorf("failed to update favorited: %w", err)
		}
		err = r.userRepo.UpdateFavorite(ctx, userId, -1)
		if err != nil {
			return fmt.Errorf("failed to update favorite: %w", err)
		}
		err = r.UpdateVideoFavorite(videoId, -1)
		if err != nil {
			return fmt.Errorf("failed to update favorite count: %w", err)
		}
		return nil
	})
	if result != nil {
		return errors.New("failed to delete favorite")
	}
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
