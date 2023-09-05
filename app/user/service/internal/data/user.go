package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/toomanysource/atreus/app/user/service/internal/server"

	"github.com/segmentio/kafka-go"

	"github.com/toomanysource/atreus/pkg/kafkaX"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"

	"github.com/toomanysource/atreus/app/user/service/internal/biz"
)

var ErrUserNotFound = errors.Join(biz.ErrUserNotFound)

var FixedCacheExpire = 720

var userTableName = "users"

// User 是用户的全量信息，包含敏感字段，是数据库的模型
type User struct {
	Id              uint32         `gorm:"primary_key"`
	Username        string         `gorm:"column:username;not null;index:idx_uname_pwd"`
	Password        string         `gorm:"column:password;not null;index:idx_uname_pwd"`
	Name            string         `gorm:"column:name;not null"`
	FollowCount     uint32         `gorm:"column:follow_count;not null;default:0"`
	FollowerCount   uint32         `gorm:"column:follower_count;not null;default:0"`
	Avatar          string         `gorm:"column:avatar_url;type:longtext;not null"`
	BackgroundImage string         `gorm:"column:background_image_url;type:longtext;not null"`
	Signature       string         `gorm:"column:signature;not null;type:longtext"`
	TotalFavorited  uint32         `gorm:"column:total_favorited;not null;default:0"`
	WorkCount       uint32         `gorm:"column:work_count;not null;default:0"`
	FavoriteCount   uint32         `grom:"column:favorite_count;not null;default:0"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (User) TableName() string {
	return userTableName
}

// UserDetail 是不包含敏感信息的用户信息
type UserDetail struct {
	Id              uint32 `gorm:"primary_key" json:"id"`
	Username        string `gorm:"column:username" json:"username"`
	Name            string `gorm:"column:name" json:"name"`
	FollowCount     uint32 `gorm:"column:follow_count" json:"follow_count"`
	FollowerCount   uint32 `gorm:"column:follower_count" json:"follower_count"`
	Avatar          string `gorm:"column:avatar_url" json:"avatar"`
	BackgroundImage string `gorm:"column:background_image_url" json:"background_image"`
	Signature       string `gorm:"column:signature" json:"signature"`
	TotalFavorited  uint32 `gorm:"column:total_favorited" json:"total_favorited"`
	WorkCount       uint32 `gorm:"column:work_count" json:"work_count"`
	FavoriteCount   uint32 `grom:"column:favorite_count" json:"favorite_count"`
}

type RelationRepo interface {
	IsFollow(ctx context.Context, userId uint32, toUserId []uint32) ([]bool, error)
}

type userRepo struct {
	db           *gorm.DB
	kfk          KfkReader
	relationRepo RelationRepo
	rdb          *redis.Client
	log          *log.Helper
}

// NewUserRepo .
func NewUserRepo(data *Data, relationConn server.RelationConn, logger log.Logger) biz.UserRepo {
	logs := log.NewHelper(log.With(logger, "data", "user_repo"))
	r := &userRepo{
		db:           data.db,
		kfk:          data.kfk,
		relationRepo: NewRelationRepo(relationConn),
		rdb:          data.rdb,
		log:          logs,
	}
	if err := r.db.AutoMigrate(&User{}); err != nil {
		log.Fatalf("database %s initialize failed: %s", userTableName, err.Error())
	}
	return r
}

// Create .
func (r *userRepo) Create(ctx context.Context, user *biz.User) (*biz.User, error) {
	u := new(User)
	copier.Copy(u, user)
	err := r.db.WithContext(ctx).Save(u).Error
	if err != nil {
		return nil, err
	}
	// 缓存用户信息
	go func() {
		uDetail := new(UserDetail)
		copier.Copy(uDetail, u)
		err := r.cacheUserDetailById(uDetail)
		if err != nil {
			r.log.Errorf("cache user by id %d failed: %s", u.Id, err.Error())
		}
	}()
	return user, nil
}

// FindById 返回的是*biz.User
func (r *userRepo) FindById(ctx context.Context, id uint32) (*biz.User, error) {
	u, err := r.findById(ctx, id)
	if err != nil {
		return nil, err
	}
	user := new(biz.User)
	copier.Copy(user, u)
	return user, nil
}

// findById 返回的是*UserDetail
func (r *userRepo) findById(ctx context.Context, id uint32) (*UserDetail, error) {
	// 先查看缓存有无对应的user信息
	uDetail, err := r.getCachedUserDetailById(ctx, id)
	if err == nil {
		return uDetail, nil
	}
	// 如果遇到错误，但不是key不存在的错误，则输出到日志里，继续查询数据库
	if !errors.Is(err, redis.Nil) {
		r.log.Errorf("get user cache by id %d failed: %s", id, err.Error())
	}
	uDetail = new(UserDetail)
	err = r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		First(uDetail).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	// 缓存用户信息
	go func() {
		err := r.cacheUserDetailById(uDetail)
		if err != nil {
			r.log.Errorf("cache user by id %d failed: %s", id, err.Error())
		}
	}()
	return uDetail, nil
}

// FindByIds .
func (r *userRepo) FindByIds(ctx context.Context, userId uint32, ids []uint32) ([]*biz.User, error) {
	result := make([]*biz.User, 0, len(ids))
	isFollow, err := r.relationRepo.IsFollow(ctx, userId, ids)
	if err != nil {
		return nil, err
	}
	// 记录查询过的id，避免出现查询重复的id
	once := make(map[uint32]int)
	session := r.db.WithContext(ctx)
	for i, id := range ids {
		// 重复id无需查询，从已查询的结果中获取
		if idx, ok := once[id]; ok {
			result[idx].IsFollow = isFollow[i]
			result = append(result, result[idx])
			continue
		}
		// 先查看缓存有无对应的user信息
		uDetail, err := r.getCachedUserDetailById(ctx, id)
		if err == nil {
			user := new(biz.User)
			copier.Copy(user, uDetail)
			user.IsFollow = isFollow[i]
			result = append(result, user)
			once[id] = len(result) - 1
			continue
		}
		// 如果遇到错误，但不是key不存在的错误，则输出到日志里，继续查询数据库
		if !errors.Is(err, redis.Nil) {
			r.log.Errorf("get user cache by id %d failed: %s", id, err.Error())
		}
		uDetail = new(UserDetail)
		// 对于唯一id，进行查询
		err = session.Model(&User{}).
			Where("id = ?", id).
			First(uDetail).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		// 缓存用户信息
		go func() {
			err := r.cacheUserDetailById(uDetail)
			if err != nil {
				r.log.Errorf("cache user by id %d failed: %s", uDetail.Id, err.Error())
			}
		}()
		user := new(biz.User)
		copier.Copy(user, uDetail)
		user.IsFollow = isFollow[i]
		result = append(result, user)
		once[id] = len(result) - 1
	}
	return result, nil
}

// FindByUsername .
func (r *userRepo) FindByUsername(ctx context.Context, username string) (*biz.User, error) {
	u, err := r.findByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	user := new(biz.User)
	copier.Copy(user, u)
	return user, nil
}

func (r *userRepo) findByUsername(ctx context.Context, username string) (*UserDetail, error) {
	uDetail := new(UserDetail)
	err := r.db.WithContext(ctx).Model(&User{}).
		Where("username = ?", username).
		First(uDetail).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return uDetail, nil
}

func (r *userRepo) InitUpdateFollowQueue() {
	kafkaX.Reader(r.kfk.follow, r.log, func(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
		userId, err := strconv.Atoi(string(msg.Key))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		change, err := strconv.Atoi(string(msg.Value))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		err = r.UpdateFollow(ctx, uint32(userId), int32(change))
		if err != nil {
			r.log.Errorf("update favorite count error, err: %v", err)
			return
		}
	})
}

func (r *userRepo) InitUpdateFollowerQueue() {
	kafkaX.Reader(r.kfk.follower, r.log, func(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
		userId, err := strconv.Atoi(string(msg.Key))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		change, err := strconv.Atoi(string(msg.Value))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		err = r.UpdateFollower(ctx, uint32(userId), int32(change))
		if err != nil {
			r.log.Errorf("update favorite count error, err: %v", err)
			return
		}
	})
}

func (r *userRepo) InitUpdateFavoriteQueue() {
	kafkaX.Reader(r.kfk.favorite, r.log, func(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
		id, err := strconv.Atoi(string(msg.Key))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		change, err := strconv.Atoi(string(msg.Value))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		err = r.UpdateFavorite(ctx, uint32(id), int32(change))
		if err != nil {
			r.log.Errorf("update favorite count error, err: %v", err)
			return
		}
	})
}

func (r *userRepo) InitUpdateFavoredQueue() {
	kafkaX.Reader(r.kfk.favored, r.log, func(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
		id, err := strconv.Atoi(string(msg.Key))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		change, err := strconv.Atoi(string(msg.Value))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		err = r.UpdateFavorited(ctx, uint32(id), int32(change))
		if err != nil {
			r.log.Errorf("update favorite count error, err: %v", err)
			return
		}
	})
}

func (r *userRepo) InitUpdatePublishQueue() {
	kafkaX.Reader(r.kfk.publish, r.log, func(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
		id, err := strconv.Atoi(string(msg.Key))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		change, err := strconv.Atoi(string(msg.Value))
		if err != nil {
			r.log.Errorf("strconv.Atoi error, err: %v", err)
			return
		}
		err = r.UpdateWork(ctx, uint32(id), int32(change))
		if err != nil {
			r.log.Errorf("update favorite count error, err: %v", err)
			return
		}
	})
}

func (r *userRepo) updateCache(user *UserDetail) error {
	err := r.cacheUserDetailById(user)
	if err != nil {
		r.log.Errorf("cache user by id %d failed: %s", user.Id, err.Error())
	}
	return nil
}

// UpdateFollow .
func (r *userRepo) UpdateFollow(ctx context.Context, id uint32, followChange int32) error {
	uDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(uDetail.FollowCount, followChange)
	uDetail.FollowCount = newValue
	err = r.updateCache(uDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("follow_count", newValue).Error
}

// UpdateFollower .
func (r *userRepo) UpdateFollower(ctx context.Context, id uint32, followerChange int32) error {
	uDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(uDetail.FollowerCount, followerChange)
	uDetail.FollowerCount = newValue
	err = r.updateCache(uDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("follower_count", newValue).Error
}

// UpdateFavorited .
func (r *userRepo) UpdateFavorited(ctx context.Context, id uint32, favoritedChange int32) error {
	uDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(uDetail.TotalFavorited, favoritedChange)
	uDetail.TotalFavorited = newValue
	err = r.updateCache(uDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("total_favorited", newValue).Error
}

// UpdateWork .
func (r *userRepo) UpdateWork(ctx context.Context, id uint32, workChange int32) error {
	uDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(uDetail.WorkCount, workChange)
	uDetail.WorkCount = newValue
	err = r.updateCache(uDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("work_count", newValue).Error
}

// UpdateFavorite .
func (r *userRepo) UpdateFavorite(ctx context.Context, id uint32, favoriteChange int32) error {
	uDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(uDetail.FavoriteCount, favoriteChange)
	uDetail.FavoriteCount = newValue
	err = r.updateCache(uDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("favorite_count", newValue).Error
}

// cacheUserById 用id来缓存User信息
func (r *userRepo) cacheUserDetailById(user *UserDetail) error {
	bs, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return r.rdb.Set(context.TODO(), getUserCachedKeyById(user.Id), bs, time.Duration(FixedCacheExpire)*time.Minute).Err()
}

// getCachedUserById 根据id来获取缓存的User信息
// error 可能是key不存在
func (r *userRepo) getCachedUserDetailById(ctx context.Context, id uint32) (*UserDetail, error) {
	user := new(UserDetail)
	bs, err := r.rdb.Get(ctx, getUserCachedKeyById(id)).Bytes()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bs, user)
	return user, err
}

// getUserCachedKeyById 获取根据id拼接成用于缓存User的key
func getUserCachedKeyById(id uint32) string {
	return fmt.Sprintf("user:id:%d", id)
}

// addUint32int32 计算uint32数字与int32数字的和，结果为uint32
// 若uint32数字的值小于int32数字的值，则返回结果为0
func addUint32int32(src uint32, mod int32) uint32 {
	if mod < 0 {
		mod = -mod
		if src < uint32(mod) {
			return 0
		}
		return src - uint32(mod)
	}
	return src + uint32(mod)
}
