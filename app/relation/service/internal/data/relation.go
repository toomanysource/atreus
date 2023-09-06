package data

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/toomanysource/atreus/pkg/errorX"

	"github.com/toomanysource/atreus/middleware"

	"github.com/toomanysource/atreus/pkg/kafkaX"

	"github.com/toomanysource/atreus/app/relation/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"
)

const (
	OccupyKey   = "-1"
	OccupyValue = ""
)

var (
	ErrFollowYourself   = errors.New("can't follow yourself")
	ErrExistRelation    = errors.New("exist relation")
	ErrNotExistRelation = errors.New("not exist relation")
)

type UserRepo interface {
	GetUserInfos(ctx context.Context, userId uint32, userIds []uint32) ([]*biz.User, error)
}

type Followers struct {
	Id         uint32 `gorm:"primary_key"`
	UserId     uint32 `gorm:"column:user_id;not null;index:idx_user_id"`
	FollowerId uint32 `gorm:"column:follower_id;not null;index:idx_follower_id"`
}

func (Followers) TableName() string {
	return "followers"
}

type relationRepo struct {
	data     *Data
	kfk      KfkWriter
	userRepo UserRepo
	log      *log.Helper
}

func NewRelationRepo(data *Data, conn *grpc.ClientConn, logger log.Logger) biz.RelationRepo {
	return &relationRepo{
		data:     data,
		kfk:      data.kfk,
		userRepo: NewUserRepo(conn),
		log:      log.NewHelper(logger),
	}
}

// Follow 关注
func (r *relationRepo) Follow(ctx context.Context, toUserId uint32) error {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	if userId == toUserId {
		return ErrFollowYourself
	}
	// 先在数据库中插入关系
	if err := r.AddFollow(ctx, userId, toUserId); err != nil {
		return err
	}
	go func() {
		ctx := context.TODO()
		if err := r.AddFollowCache(ctx, userId, toUserId); err != nil {
			r.log.Error(err)
			return
		}
		r.log.Info("redis store success")
	}()
	go func() {
		ctx := context.TODO()
		if err := r.AddFollowedCache(ctx, toUserId, userId); err != nil {
			r.log.Error(err)
			return
		}
		r.log.Info("redis store success")
	}()
	r.log.Infof(
		"CreateRelation -> userId: %v - toUserId: %v", userId, toUserId)
	return nil
}

// UnFollow 取消关注
func (r *relationRepo) UnFollow(ctx context.Context, toUserId uint32) error {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	err := r.DelFollow(ctx, userId, toUserId)
	if err != nil {
		return err
	}
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.CheckFollowCache(ctx, userId, toUserId)
		if err != nil {
			r.log.Error(err)
			return
		}
		if ok {
			// 如果存在则删除
			if err = r.DeleteFollowCache(ctx, userId, toUserId); err != nil {
				r.log.Error(err)
				return
			}
		}
	}()
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.CheckFollowedCache(ctx, toUserId, userId)
		if err != nil {
			r.log.Error(err)
			return
		}
		if ok {
			// 如果存在则删除
			if err = r.DeleteFollowedCache(ctx, toUserId, userId); err != nil {
				r.log.Error(err)
				return
			}
		}
	}()
	r.log.Infof(
		"DelRelation -> userId: %v - toUserId: %v", userId, toUserId)
	return nil
}

// IsFollow 查询是否关注
func (r *relationRepo) IsFollow(ctx context.Context, userId uint32, toUserId []uint32) (oks []bool, err error) {
	count, err := r.data.cache.followRelation.Exists(ctx, strconv.Itoa(int(userId))).Result()
	if err != nil {
		return nil, errors.Join(errorX.ErrRedisQuery, err)
	}
	if count > 0 {
		once := make(map[uint32]bool)
		for _, v := range toUserId {
			if _, ok := once[v]; ok {
				oks = append(oks, once[v])
				continue
			}
			ok, err := r.CheckFollowCache(ctx, userId, v)
			if err != nil {
				return nil, err
			}
			once[v] = ok
			oks = append(oks, ok)
		}
		return oks, nil
	}
	go func() {
		// 如果不存在则创建
		cl, err := r.GetFlList(ctx, userId)
		if err != nil {
			r.log.Error(err)
			return
		}
		if err = CreateCacheByTran(ctx, r.data.cache.followRelation, cl, userId); err != nil {
			r.log.Error(err)
			return
		}
		r.log.Info("redis transaction success")
	}()
	return r.SearchRelation(ctx, userId, toUserId)
}

// GetFollowList 获取关注列表
func (r *relationRepo) GetFollowList(ctx context.Context, userId uint32) ([]*biz.User, error) {
	userID := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	// 先在redis缓存中查询是否存在关注列表
	follows, err := r.GetFollowCache(ctx, userId)
	if err != nil {
		return nil, err
	}
	fl := make([]uint32, 0, len(follows))
	if len(follows) > 0 {
		for _, v := range follows {
			if v == OccupyKey {
				continue
			}
			vc, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			fl = append(fl, uint32(vc))
		}
	} else {
		// 如果不存在则创建
		fl, err = r.GetFlList(ctx, userId)
		if err != nil {
			return nil, err
		}
		// 将关注列表存入redis缓存
		go func(l []uint32) {
			if err = CreateCacheByTran(context.Background(), r.data.cache.followRelation, l, userId); err != nil {
				r.log.Error(err)
				return
			}
			r.log.Info("redis transaction success")
		}(fl)
	}
	if len(fl) == 0 {
		return nil, nil
	}
	if userID != userId {
		// 查询是否关注
		isFollow, err := r.SearchRelation(ctx, userID, fl)
		if err != nil {
			return nil, err
		}
		users, err := r.userRepo.GetUserInfos(ctx, userID, fl)
		if err != nil {
			return nil, err
		}
		for i, user := range users {
			user.IsFollow = isFollow[i]
		}
		r.log.Infof(
			"GetFollowUserList -> userId: %v - userList: %v", userId, users)
		return users, nil
	}
	users, err := r.userRepo.GetUserInfos(ctx, userId, fl)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		user.IsFollow = true
	}
	r.log.Infof(
		"GetFollowUserList -> userId: %v - userList: %v", userId, users)
	return users, nil
}

// GetFollowerList 获取粉丝列表
func (r *relationRepo) GetFollowerList(ctx context.Context, userId uint32) (ul []*biz.User, err error) {
	// 先在redis缓存中查询是否存在被关注列表
	userID := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	followers, err := r.GetFollowedCache(ctx, userId)
	if err != nil {
		return nil, err
	}
	fl := make([]uint32, 0, len(followers))
	if len(followers) > 0 {
		for _, v := range followers {
			if v == OccupyKey {
				continue
			}
			vc, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			fl = append(fl, uint32(vc))
		}
	} else {
		// 如果不存在则创建
		fl, err = r.GetFlrList(ctx, userId)
		if err != nil {
			return nil, err
		}
		// 将关注列表存入redis缓存
		go func(l []uint32) {
			if err = CreateCacheByTran(context.Background(), r.data.cache.followedRelation, l, userId); err != nil {
				r.log.Error(err)
				return
			}
			r.log.Info("redis transaction success")
		}(fl)
	}
	if len(fl) == 0 {
		return nil, nil
	}
	if userID != userId {
		// 查询是否关注
		isFollow, err := r.SearchRelation(ctx, userID, fl)
		if err != nil {
			return nil, err
		}
		users, err := r.userRepo.GetUserInfos(ctx, userID, fl)
		if err != nil {
			return nil, err
		}
		for i, user := range users {
			user.IsFollow = isFollow[i]
		}
		r.log.Infof(
			"GetFollowUserList -> userId: %v - userList: %v", userId, users)
		return users, nil
	}
	// 查询是否关注粉丝
	isFollow, err := r.SearchRelation(ctx, userId, fl)
	if err != nil {
		return nil, err
	}
	users, err := r.userRepo.GetUserInfos(ctx, userId, fl)
	if err != nil {
		return nil, err
	}
	for i, user := range users {
		user.IsFollow = isFollow[i]
	}
	r.log.Infof(
		"GetFollowUserList -> userId: %v - userList: %v", userId, users)
	return users, nil
}

// GetFollowCache 获取关注缓存
func (r *relationRepo) GetFollowCache(ctx context.Context, userId uint32) ([]string, error) {
	follows, err := r.data.cache.followRelation.HKeys(ctx, strconv.Itoa(int(userId))).Result()
	if err != nil {
		return nil, errors.Join(errorX.ErrRedisQuery, err)
	}
	return follows, nil
}

// AddFollowCache 添加关注缓存
func (r *relationRepo) AddFollowCache(ctx context.Context, userId uint32, toUserId uint32) error {
	// 在redis缓存中查询是否存在
	ok, err := r.CheckFollowCache(ctx, userId, toUserId)
	if err != nil {
		return err
	}
	if !ok {
		// 如果不存在则创建
		cl, err := r.GetFlList(ctx, userId)
		if err != nil {
			return err
		}
		// 没有关注列表则不创建
		if len(cl) == 0 {
			return err
		}
		return CreateCacheByTran(ctx, r.data.cache.followRelation, cl, userId)
	}
	return r.CreateFollowCache(ctx, userId, toUserId)
}

// CheckFollowCache 查询关注缓存
func (r *relationRepo) CheckFollowCache(ctx context.Context, userId uint32, toUserId uint32) (bool, error) {
	ok, err := r.data.cache.followRelation.HExists(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(toUserId))).Result()
	if err != nil {
		return false, errors.Join(errorX.ErrRedisQuery, err)
	}
	return ok, nil
}

// CreateFollowCache 创建关注缓存
func (r *relationRepo) CreateFollowCache(ctx context.Context, userId uint32, toUserId uint32) (err error) {
	if err = r.data.cache.followRelation.HSet(
		ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(toUserId)), "").Err(); err != nil {
		return errors.Join(errorX.ErrRedisSet, err)
	}
	return nil
}

// DeleteFollowCache 删除关注缓存
func (r *relationRepo) DeleteFollowCache(ctx context.Context, userId uint32, toUserId uint32) error {
	if err := r.data.cache.followRelation.HDel(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(toUserId))).Err(); err != nil {
		return errors.Join(errorX.ErrRedisDelete, err)
	}
	return nil
}

// GetFollowedCache 获取粉丝缓存
func (r *relationRepo) GetFollowedCache(ctx context.Context, userId uint32) ([]string, error) {
	followers, err := r.data.cache.followedRelation.HKeys(ctx, strconv.Itoa(int(userId))).Result()
	if err != nil {
		return nil, errors.Join(errorX.ErrRedisQuery, err)
	}
	return followers, nil
}

// AddFollowedCache 添加粉丝缓存
func (r *relationRepo) AddFollowedCache(ctx context.Context, toUserId uint32, userId uint32) (err error) {
	// 在redis缓存中查询是否存在
	ok, err := r.CheckFollowedCache(ctx, toUserId, userId)
	if err != nil {
		return err
	}
	if !ok {
		// 如果不存在则创建
		cl, err := r.GetFlrList(ctx, toUserId)
		if err != nil {
			return err
		}
		return CreateCacheByTran(ctx, r.data.cache.followedRelation, cl, toUserId)
	}
	return r.CreateFollowedCache(ctx, toUserId, userId)
}

// CheckFollowedCache 查询粉丝缓存
func (r *relationRepo) CheckFollowedCache(ctx context.Context, toUserId uint32, userId uint32) (bool, error) {
	ok, err := r.data.cache.followedRelation.HExists(ctx, strconv.Itoa(int(toUserId)), strconv.Itoa(int(userId))).Result()
	if err != nil {
		return false, errors.Join(errorX.ErrRedisQuery, err)
	}
	return ok, nil
}

// CreateFollowedCache 创建粉丝缓存
func (r *relationRepo) CreateFollowedCache(ctx context.Context, toUserId uint32, userId uint32) (err error) {
	if err = r.data.cache.followedRelation.HSet(
		ctx, strconv.Itoa(int(toUserId)), strconv.Itoa(int(userId)), "").Err(); err != nil {
		return errors.Join(errorX.ErrRedisSet, err)
	}
	return nil
}

// DeleteFollowedCache 删除粉丝缓存
func (r *relationRepo) DeleteFollowedCache(ctx context.Context, toUserId uint32, userId uint32) error {
	if err := r.data.cache.followedRelation.HDel(ctx, strconv.Itoa(int(toUserId)), strconv.Itoa(int(userId))).Err(); err != nil {
		return errors.Join(errorX.ErrRedisDelete, err)
	}
	return nil
}

// GetFlList 数据库获取关注列表
func (r *relationRepo) GetFlList(ctx context.Context, userId uint32) (userIDs []uint32, err error) {
	var follows []*Followers
	if err := r.data.db.WithContext(ctx).Where("follower_id = ?", userId).Find(&follows).Error; err != nil {
		return nil, errors.Join(errorX.ErrMysqlQuery, err)
	}
	if len(follows) == 0 {
		return nil, nil
	}
	for _, follow := range follows {
		userIDs = append(userIDs, follow.UserId)
	}
	return
}

// GetFlrList 数据库获取粉丝列表
func (r *relationRepo) GetFlrList(ctx context.Context, userId uint32) ([]uint32, error) {
	var followers []*Followers
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userId).Find(&followers).Error; err != nil {
		return nil, errors.Join(errorX.ErrMysqlQuery, err)
	}
	if len(followers) == 0 {
		return nil, nil
	}
	userIDs := make([]uint32, 0, len(followers))
	for _, follower := range followers {
		userIDs = append(userIDs, follower.FollowerId)
	}
	return userIDs, nil
}

// AddFollow 数据库添加关注关系
func (r *relationRepo) AddFollow(ctx context.Context, userId uint32, toUserId uint32) error {
	follow := &Followers{
		UserId:     toUserId,
		FollowerId: userId,
	}
	result := r.data.db.WithContext(ctx).FirstOrCreate(&follow)
	if result.RowsAffected == 0 {
		return ErrExistRelation
	}
	if result.Error != nil {
		return errors.Join(errorX.ErrMysqlInsert, result.Error)
	}
	go func() {
		err := kafkaX.Update(r.kfk.follow, strconv.Itoa(int(userId)), "1")
		if err != nil {
			r.log.Error(err)
		}
	}()
	go func() {
		err := kafkaX.Update(r.kfk.follower, strconv.Itoa(int(toUserId)), "1")
		if err != nil {
			r.log.Error(err)
		}
	}()
	return nil
}

// DelFollow 数据库取消关注关系
func (r *relationRepo) DelFollow(ctx context.Context, userId uint32, toUserId uint32) error {
	result := r.data.db.WithContext(ctx).Where(
		"user_id = ? AND follower_id = ?", toUserId, userId).Delete(&Followers{})
	if result.Error != nil {
		return errors.Join(errorX.ErrMysqlDelete, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotExistRelation
	}
	go func() {
		err := kafkaX.Update(r.kfk.follow, strconv.Itoa(int(userId)), "-1")
		if err != nil {
			r.log.Error(err)
		}
	}()
	go func() {
		err := kafkaX.Update(r.kfk.follower, strconv.Itoa(int(toUserId)), "-1")
		if err != nil {
			r.log.Error(err)
		}
	}()
	return nil
}

// SearchRelation 数据库查询关注关系
func (r *relationRepo) SearchRelation(ctx context.Context, userId uint32, toUserId []uint32) ([]bool, error) {
	var relation []*Followers
	relationMap := make(map[uint32]bool, len(relation))
	if err := r.data.db.WithContext(ctx).
		Where("user_id IN ? AND follower_id = ?", toUserId, userId).
		Find(&relation).Error; err != nil {
		return nil, errors.Join(errorX.ErrMysqlQuery, err)
	}
	for _, follow := range relation {
		relationMap[follow.UserId] = true
	}
	slice := make([]bool, 0, len(toUserId))
	for _, id := range toUserId {
		if _, ok := relationMap[id]; !ok {
			slice = append(slice, false)
			continue
		}
		slice = append(slice, true)
	}
	return slice, nil
}

// CreateCacheByTran 缓存创建事务
func CreateCacheByTran(ctx context.Context, cache *redis.Client, ul []uint32, userId uint32) error {
	// 使用事务将列表存入redis缓存
	_, err := cache.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		insertMap := make(map[string]interface{}, len(ul))
		insertMap[OccupyKey] = OccupyValue
		for _, v := range ul {
			vs := strconv.Itoa(int(v))
			insertMap[vs] = OccupyValue
		}
		err := pipe.HMSet(ctx, strconv.Itoa(int(userId)), insertMap).Err()
		if err != nil {
			return errors.Join(errorX.ErrRedisSet, err)
		}
		// 将评论数量存入redis缓存,使用随机过期时间防止缓存雪崩
		begin, end := 360, 720
		err = pipe.Expire(ctx, strconv.Itoa(int(userId)), randomTime(time.Minute, begin, end)).Err()
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
