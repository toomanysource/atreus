package data

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/toomanysource/atreus/app/relation/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type UserRepo interface {
	GetUserInfos(ctx context.Context, userId uint32, userIds []uint32) ([]*biz.User, error)
	UpdateFollow(ctx context.Context, userId uint32, followChange int32) error
	UpdateFollower(ctx context.Context, userId uint32, followerChange int32) error
}

type Followers struct {
	Id         uint32 `gorm:"primary_key"`
	UserId     uint32 `gorm:"column:user_id;not null"`
	FollowerId uint32 `gorm:"column:follower_id;not null"`
}

func (Followers) TableName() string {
	return "followers"
}

type relationRepo struct {
	data     *Data
	userRepo UserRepo
	log      *log.Helper
}

func NewRelationRepo(data *Data, conn *grpc.ClientConn, logger log.Logger) biz.RelationRepo {
	return &relationRepo{
		data:     data,
		userRepo: NewUserRepo(conn),
		log:      log.NewHelper(logger),
	}
}

func (r *relationRepo) GetFollowList(ctx context.Context, userId uint32) ([]*biz.User, error) {
	// 先在redis缓存中查询是否存在关注列表
	follows, err := r.data.cache.followRelation.HKeys(ctx, strconv.Itoa(int(userId))).Result()
	if err != nil {
		return nil, fmt.Errorf("redis query error %w", err)
	}
	fl := make([]uint32, 0, len(follows))
	if len(follows) > 0 {
		for _, v := range follows {
			vc, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("strconv error %w", err)
			}
			fl = append(fl, uint32(vc))
		}
	} else {
		// 如果不存在则创建
		fl, err = r.GetFlList(ctx, userId)
		if err != nil {
			return nil, err
		}
		// 没有关注列表则不创建
		if len(fl) == 0 {
			return nil, nil
		}
		// 将关注列表存入redis缓存
		go func(l []uint32) {
			if err = CacheCreateRelationTransaction(context.Background(), r.data.cache.followRelation, l, userId); err != nil {
				r.log.Errorf("redis transaction error %w", err)
				return
			}
			r.log.Info("redis transaction success")
		}(fl)
	}
	users, err := r.userRepo.GetUserInfos(ctx, 0, fl)
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

func (r *relationRepo) GetFollowerList(ctx context.Context, userId uint32) (ul []*biz.User, err error) {
	// 先在redis缓存中查询是否存在被关注列表
	followers, err := r.data.cache.followedRelation.HKeys(ctx, strconv.Itoa(int(userId))).Result()
	if err != nil {
		return nil, fmt.Errorf("redis query error %w", err)
	}
	fl := make([]uint32, 0, len(followers))
	if len(followers) > 0 {
		for _, v := range followers {
			vc, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("strconv error %w", err)
			}
			fl = append(fl, uint32(vc))
		}
	} else {
		// 如果不存在则创建
		fl, err = r.GetFlrList(ctx, userId)
		if err != nil {
			return nil, err
		}
		// 没有粉丝列表则不创建
		if len(fl) == 0 {
			return nil, nil
		}
		// 将关注列表存入redis缓存
		go func(l []uint32) {
			if err = CacheCreateRelationTransaction(context.Background(), r.data.cache.followedRelation, l, userId); err != nil {
				r.log.Errorf("redis transaction error %w", err)
				return
			}
			r.log.Info("redis transaction success")
		}(fl)
	}
	// 查询是否关注粉丝
	isFollow, err := r.SearchRelation(ctx, userId, fl)
	if err != nil {
		return nil, err
	}
	users, err := r.userRepo.GetUserInfos(ctx, 0, fl)
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

func (r *relationRepo) Follow(ctx context.Context, toUserId uint32) error {
	userId := ctx.Value("user_id").(uint32)
	if userId == toUserId {
		return fmt.Errorf("can't follow yourself")
	}
	// 先在数据库中插入关系
	err := r.AddFollow(ctx, userId, toUserId)
	if err != nil {
		return err
	}
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.data.cache.followRelation.HExists(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(toUserId))).Result()
		if err != nil {
			r.log.Errorf("redis query error %w", err)
			return
		}
		if !ok {
			// 如果不存在则创建
			cl, err := r.GetFlList(ctx, userId)
			if err != nil {
				r.log.Errorf("mysql query error %w", err)
				return
			}
			// 没有关注列表则不创建
			if len(cl) == 0 {
				return
			}
			if err = CacheCreateRelationTransaction(ctx, r.data.cache.followRelation, cl, userId); err != nil {
				r.log.Errorf("redis transaction error %w", err)
				return
			}
			r.log.Info("redis transaction success")
			return
		}
		if err = r.data.cache.followRelation.HSet(
			ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(toUserId)), "").Err(); err != nil {
			r.log.Errorf("redis store error %w", err)
			return
		}
		r.log.Info("redis store success")
	}()
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.data.cache.followedRelation.HExists(ctx, strconv.Itoa(int(toUserId)), strconv.Itoa(int(userId))).Result()
		if err != nil {
			r.log.Errorf("redis query error %w", err)
			return
		}
		if !ok {
			// 如果不存在则创建
			cl, err := r.GetFlrList(ctx, toUserId)
			if err != nil {
				r.log.Errorf("mysql query error %w", err)
				return
			}
			// 没有粉丝列表则不创建
			if len(cl) == 0 {
				return
			}
			if err = CacheCreateRelationTransaction(ctx, r.data.cache.followedRelation, cl, toUserId); err != nil {
				r.log.Errorf("redis transaction error %w", err)
				return
			}
			r.log.Info("redis transaction success")
			return
		}
		if err = r.data.cache.followedRelation.HSet(
			ctx, strconv.Itoa(int(toUserId)), strconv.Itoa(int(userId)), "").Err(); err != nil {
			r.log.Errorf("redis store error %w", err)
			return
		}
		r.log.Info("redis store success")
	}()
	r.log.Infof(
		"CreateRelation -> userId: %v - toUserId: %v", userId, toUserId)
	return nil
}

func (r *relationRepo) UnFollow(ctx context.Context, toUserId uint32) error {
	userId := ctx.Value("user_id").(uint32)
	err := r.DelFollow(ctx, userId, toUserId)
	if err != nil {
		return err
	}
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.data.cache.followRelation.HExists(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(toUserId))).Result()
		if err != nil {
			r.log.Errorf("redis query error %w", err)
			return
		}
		if ok {
			// 如果存在则删除
			if err = r.data.cache.followRelation.HDel(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(toUserId))).Err(); err != nil {
				r.log.Errorf("redis delete error %w", err)
				return
			}
		}
	}()
	go func() {
		ctx := context.TODO()
		// 在redis缓存中查询是否存在
		ok, err := r.data.cache.followedRelation.HExists(ctx, strconv.Itoa(int(toUserId)), strconv.Itoa(int(userId))).Result()
		if err != nil {
			r.log.Errorf("redis query error %w", err)
			return
		}
		if ok {
			// 如果存在则删除
			if err = r.data.cache.followedRelation.HDel(ctx, strconv.Itoa(int(toUserId)), strconv.Itoa(int(userId))).Err(); err != nil {
				r.log.Errorf("redis delete error %w", err)
				return
			}
		}
	}()
	r.log.Infof(
		"DelRelation -> userId: %v - toUserId: %v", userId, toUserId)
	return nil
}

func (r *relationRepo) IsFollow(ctx context.Context, userId uint32, toUserId []uint32) (oks []bool, err error) {
	count, err := r.data.cache.followRelation.Exists(ctx, strconv.Itoa(int(userId))).Result()
	if err != nil {
		return nil, fmt.Errorf("redis query error %w", err)
	}
	if count > 0 {
		for _, v := range toUserId {
			ok, err := r.data.cache.followRelation.HExists(ctx, strconv.Itoa(int(userId)), strconv.Itoa(int(v))).Result()
			if err != nil {
				return nil, fmt.Errorf("redis query error %w", err)
			}
			oks = append(oks, ok)
		}
		return oks, nil
	}
	return r.SearchRelation(ctx, userId, toUserId)
}

// GetFlList 数据库获取关注列表
func (r *relationRepo) GetFlList(ctx context.Context, userId uint32) (userIDs []uint32, err error) {
	var follows []*Followers
	result := r.data.db.WithContext(ctx).Where("follower_id = ?", userId).Find(&follows)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
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
	result := r.data.db.WithContext(ctx).Where("user_id = ?", userId).Find(&followers)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	userIDs := make([]uint32, 0, len(followers))
	for _, follower := range followers {
		userIDs = append(userIDs, follower.FollowerId)
	}
	return userIDs, nil
}

// AddFollow 添加关注
func (r *relationRepo) AddFollow(ctx context.Context, userId uint32, toUserId uint32) error {
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		relation, err := r.SearchRelation(ctx, userId, []uint32{toUserId})
		if err != nil {
			return fmt.Errorf("failed to search relation: %w", err)
		}
		if relation != nil {
			return nil
		}
		follow := &Followers{
			UserId:     toUserId,
			FollowerId: userId,
		}
		err = tx.Create(&follow).Error
		if err != nil {
			return fmt.Errorf("failed to create relation: %w", err)
		}
		err = r.userRepo.UpdateFollow(ctx, userId, 1)
		if err != nil {
			return fmt.Errorf("failed to update follow: %w", err)
		}
		err = r.userRepo.UpdateFollower(ctx, toUserId, 1)
		if err != nil {
			return fmt.Errorf("failed to update follower: %w", err)
		}
		return nil
	})
	return err
}

// DelFollow 取消关注
func (r *relationRepo) DelFollow(ctx context.Context, userId uint32, toUserId uint32) error {
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		relation, err := r.SearchRelation(ctx, userId, []uint32{toUserId})
		if err != nil {
			return err
		}
		if relation == nil {
			return nil
		}
		err = tx.Where(
			"user_id = ? AND follower_id = ?", toUserId, userId).Delete(&relation[0]).Error
		if err != nil {
			return err
		}
		err = r.userRepo.UpdateFollow(ctx, userId, -1)
		if err != nil {
			return err
		}
		err = r.userRepo.UpdateFollower(ctx, toUserId, -1)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

// SearchRelation 查询关注关系
func (r *relationRepo) SearchRelation(ctx context.Context, userId uint32, toUserId []uint32) ([]bool, error) {
	var relation []*Followers
	relationMap := make(map[uint32]bool, len(relation))
	result := r.data.db.WithContext(ctx).Where("user_id IN ? AND follower_id = ?", toUserId, userId).Find(&relation)
	if result.Error != nil {
		return nil, result.Error
	}
	for _, follow := range relation {
		relationMap[follow.UserId] = true
	}
	slice := make([]bool, len(toUserId))
	for _, id := range toUserId {
		if _, ok := relationMap[id]; !ok {
			slice = append(slice, false)
			continue
		}
		slice = append(slice, true)
	}
	return slice, nil
}

// CacheCreateRelationTransaction 缓存创建事务
func CacheCreateRelationTransaction(ctx context.Context, cache *redis.Client, ul []uint32, userId uint32) error {
	// 使用事务将列表存入redis缓存
	_, err := cache.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		insertMap := make(map[string]interface{}, len(ul))
		for _, v := range ul {
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
