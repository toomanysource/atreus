package data

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/toomanysource/atreus/pkg/errorX"

	"github.com/toomanysource/atreus/pkg/kafkaX"

	"github.com/go-redis/redis/v8"

	"github.com/toomanysource/atreus/app/favorite/service/internal/biz"
	"github.com/toomanysource/atreus/app/favorite/service/internal/server"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	OccupyKey   = "-1"
	OccupyValue = ""
)

var (
	ErrExistFavorite    = errors.New("exist favorite relation")
	ErrNotExistFavorite = errors.New("not exist favorite relation")
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
		log:         log.NewHelper(log.With(logger, "module", "data/favorite")),
	}
}

// CreateFavorite 创建喜爱关系
func (r *favoriteRepo) CreateFavorite(ctx context.Context, userId, videoId uint32) error {
	// 先在数据库中插入关系
	err := r.InsertFavorite(ctx, userId, videoId)
	if err != nil {
		return err
	}
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.CheckKey(ctx, userId)
		if err != nil {
			r.log.Error(err)
			return
		}
		if ok {
			// 如果存在则更新
			err = r.InsertCache(ctx, userId, videoId)
			if err != nil {
				r.log.Error(err)
				return
			}
			r.log.Info("redis store success")
		}
	}()
	r.log.Infof(
		"CreateFavorite -> userId: %v - videoId: %v", userId, videoId)
	return nil
}

// DeleteFavorite 删除喜爱关系
func (r *favoriteRepo) DeleteFavorite(ctx context.Context, userId, videoId uint32) error {
	err := r.DelFavorite(ctx, userId, videoId)
	if err != nil {
		return err
	}
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.CheckHKey(ctx, userId, videoId)
		if err != nil {
			r.log.Error(err)
			return
		}
		if ok {
			// 如果存在则删除
			err = r.DeleteCache(ctx, userId, videoId)
			if err != nil {
				r.log.Error(err)
				return
			}
		}
	}()
	r.log.Infof(
		"DeleteFavorite -> userId: %v - videoId: %v", userId, videoId)
	return nil
}

// GetFavoriteList 获取喜爱列表
func (r *favoriteRepo) GetFavoriteList(ctx context.Context, userID uint32) ([]biz.Video, error) {
	// 先在redis缓存中查询是否存在喜爱列表
	ok, err := r.CheckKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	var fl []uint32
	if ok {
		// 如果存在则获取
		fl, err = r.GetFavoritesCache(ctx, userID)
		if err != nil {
			return nil, err
		}
	} else {
		// 如果不存在则创建
		fl, err = r.GetFavoritesByUserId(ctx, userID)
		if err != nil {
			return nil, err
		}
		// 将喜爱列表存入redis缓存
		go func(l []uint32) {
			if err = CreateCacheByTran(context.Background(), r.data.cache, l, userID); err != nil {
				r.log.Error(err)
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

// IsFavorite 判断是否喜爱
func (r *favoriteRepo) IsFavorite(ctx context.Context, userId uint32, videoIds []uint32) (oks []bool, err error) {
	ok, err := r.CheckKey(ctx, userId)
	if err != nil {
		return nil, err
	}
	if ok {
		once := make(map[uint32]bool)
		for _, v := range videoIds {
			if _, ok := once[v]; ok {
				oks = append(oks, once[v])
				continue
			}
			ok, err := r.CheckHKey(ctx, userId, v)
			if err != nil {
				return nil, err
			}
			once[v] = ok
			oks = append(oks, ok)
		}
		return oks, nil
	}
	go func() {
		ctx := context.TODO()
		// 如果不存在则创建
		fl, err := r.GetFavoritesByUserId(ctx, userId)
		if err != nil {
			r.log.Error(err)
			return
		}
		// 将喜爱列表存入redis缓存
		if err = CreateCacheByTran(ctx, r.data.cache, fl, userId); err != nil {
			r.log.Error(err)
			return
		}
		r.log.Info("redis transaction success")
	}()
	return r.CheckFavorite(ctx, userId, videoIds)
}

// GetAuthorId 获取视频作者ID
func (r *favoriteRepo) GetAuthorId(ctx context.Context, userId uint32, videoId uint32) (uint32, error) {
	videoList, err := r.publishRepo.GetVideoListByVideoIds(ctx, userId, []uint32{videoId})
	if err != nil {
		return 0, err
	}

	authorId := videoList[0].Author.Id
	return authorId, nil
}

// CheckKey 检查Key缓存
func (r *favoriteRepo) CheckKey(ctx context.Context, userId uint32) (ok bool, err error) {
	// 在redis缓存中查询是否存在
	count, err := r.data.cache.Exists(ctx, strconv.Itoa(int(userId))).Result()
	if err != nil {
		return false, errors.Join(errorX.ErrRedisQuery, err)
	}
	return count > 0, nil
}

// CheckHKey 检查Hash型Key缓存
func (r *favoriteRepo) CheckHKey(ctx context.Context, userId, videoId uint32) (bool, error) {
	ok, err := r.data.cache.HExists(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(videoId))).Result()
	if err != nil {
		return false, errors.Join(errorX.ErrRedisQuery, err)
	}
	return ok, nil
}

// InsertCache 插入缓存
func (r *favoriteRepo) InsertCache(ctx context.Context, userId, videoId uint32) error {
	if err := r.data.cache.HSet(
		ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(videoId)), "").Err(); err != nil {
		return errors.Join(errorX.ErrRedisSet, err)
	}
	return nil
}

// DeleteCache 删除缓存
func (r *favoriteRepo) DeleteCache(ctx context.Context, userId, videoId uint32) (err error) {
	if err = r.data.cache.HDel(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(videoId))).Err(); err != nil {
		return errors.Join(errorX.ErrRedisDelete, err)
	}
	return nil
}

// GetFavoritesCache 获取缓存
func (r *favoriteRepo) GetFavoritesCache(ctx context.Context, userID uint32) (fl []uint32, err error) {
	favorites, err := r.data.cache.HKeys(ctx, strconv.Itoa(int(userID))).Result()
	if err != nil {
		return nil, errors.Join(errorX.ErrRedisQuery, err)
	}
	for _, v := range favorites {
		if v == OccupyKey {
			continue
		}
		vc, err := strconv.Atoi(v)
		if err != nil {
			return nil, errors.Join(errorX.ErrStrconvParse, err)
		}
		fl = append(fl, uint32(vc))
	}
	return fl, nil
}

// InsertFavorite 数据库插入喜爱关系
func (r *favoriteRepo) InsertFavorite(ctx context.Context, userId, videoId uint32) error {
	authorId, err := r.GetAuthorId(ctx, userId, videoId)
	if err != nil {
		return err
	}
	err = r.data.db.WithContext(ctx).
		Where("user_id = ? AND video_id = ?", userId, videoId).
		First(&Favorite{}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result := r.data.db.WithContext(ctx).Create(&Favorite{
			UserID:  userId,
			VideoID: videoId,
		})
		if result.Error != nil {
			return errors.Join(errorX.ErrMysqlInsert, result.Error)
		}
		go func() {
			if err = kafkaX.Update(r.kfk.Favored, strconv.Itoa(int(authorId)), "1"); err != nil {
				r.log.Error(err)
			}
		}()
		go func() {
			if err = kafkaX.Update(r.kfk.Favorite, strconv.Itoa(int(userId)), "1"); err != nil {
				r.log.Error(err)
			}
		}()
		go func() {
			if err = kafkaX.Update(r.kfk.videoFavorite, strconv.Itoa(int(videoId)), "1"); err != nil {
				r.log.Error(err)
			}
		}()
		return nil
	}
	if err != nil {
		return errors.Join(errorX.ErrMysqlQuery, err)
	}
	return ErrExistFavorite
}

// DelFavorite 数据库删除喜爱关系
func (r *favoriteRepo) DelFavorite(ctx context.Context, userId, videoId uint32) error {
	authorId, err := r.GetAuthorId(ctx, userId, videoId)
	if err != nil {
		return err
	}
	err = r.data.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userId, videoId).First(&Favorite{}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotExistFavorite
	}
	if err != nil {
		return errors.Join(errorX.ErrMysqlQuery, err)
	}
	result := r.data.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userId, videoId).Delete(&Favorite{})
	if result.Error != nil {
		return errors.Join(errorX.ErrMysqlDelete, result.Error)
	}
	go func() {
		if err = kafkaX.Update(r.kfk.Favored, strconv.Itoa(int(authorId)), "-1"); err != nil {
			r.log.Error(err)
		}
	}()
	go func() {
		if err = kafkaX.Update(r.kfk.Favorite, strconv.Itoa(int(userId)), "-1"); err != nil {
			r.log.Error(err)
		}
	}()
	go func() {
		if err = kafkaX.Update(r.kfk.videoFavorite, strconv.Itoa(int(videoId)), "-1"); err != nil {
			r.log.Error(err)
		}
	}()
	return nil
}

// GetFavoritesByUserId 数据库获取喜爱列表
func (r *favoriteRepo) GetFavoritesByUserId(ctx context.Context, userID uint32) ([]uint32, error) {
	var favorites []Favorite
	err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&favorites).Error
	if err != nil {
		return nil, errors.Join(errorX.ErrMysqlQuery, err)
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

// CheckFavorite 检查数据库是否存在喜爱关系
func (r *favoriteRepo) CheckFavorite(ctx context.Context, userId uint32, videoIds []uint32) ([]bool, error) {
	var favorites []Favorite
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND video_id IN ?", userId, videoIds).
		Find(&favorites).Error; err != nil {
		return nil, errors.Join(errorX.ErrMysqlQuery, err)
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

// CreateCacheByTran 缓存创建事务
func CreateCacheByTran(ctx context.Context, cache *redis.Client, vl []uint32, userId uint32) error {
	// 使用事务将列表存入redis缓存
	_, err := cache.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		insertMap := make(map[string]interface{}, len(vl))
		// 如果喜爱列表为空则插入一个空值,防止缓存击穿
		insertMap[OccupyKey] = OccupyValue
		for _, v := range vl {
			vs := strconv.Itoa(int(v))
			insertMap[vs] = OccupyValue
		}
		if err := pipe.HMSet(ctx, strconv.Itoa(int(userId)), insertMap).Err(); err != nil {
			return errors.Join(errorX.ErrRedisSet, err)
		}
		// 将评论数量存入redis缓存,使用随机过期时间防止缓存雪崩
		begin, end := 360, 720
		if err := pipe.Expire(ctx, strconv.Itoa(int(userId)), randomTime(time.Minute, begin, end)).Err(); err != nil {
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
