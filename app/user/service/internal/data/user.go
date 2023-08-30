package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

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

type User struct {
	Id              uint32         `gorm:"primary_key" json:"id" copier:"UserId"`
	Username        string         `gorm:"column:username;not null" json:"username"`
	Password        string         `gorm:"column:password;not null" json:"password"`
	Name            string         `gorm:"column:name;not null" json:"name"`
	FollowCount     uint32         `gorm:"column:follow_count;not null;default:0" json:"follow_count"`
	FollowerCount   uint32         `gorm:"column:follower_count;not null;default:0" json:"follower_count"`
	Avatar          string         `gorm:"column:avatar_url;type:longtext;not null" json:"avatar"`
	BackgroundImage string         `gorm:"column:background_image_url;type:longtext;not null" json:"background_image"`
	Signature       string         `gorm:"column:signature;not null;type:longtext" json:"signature"`
	TotalFavorited  uint32         `gorm:"column:total_favorited;not null;default:0" json:"total_favorited"`
	WorkCount       uint32         `gorm:"column:work_count;not null;default:0" json:"work_count"`
	FavoriteCount   uint32         `grom:"column:favorite_count;not null;default:0" json:"favorite_count"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (User) TableName() string {
	return userTableName
}

type userRepo struct {
	db  *gorm.DB
	kfk KfkReader
	rdb *redis.Client
	log *log.Helper
}

// NewUserRepo .
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	logs := log.NewHelper(log.With(logger, "data", "user_repo"))
	r := &userRepo{
		db:  data.db,
		kfk: data.kfk,
		rdb: data.rdb,
		log: logs,
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
		err := r.cacheUserById(u)
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

// findById 返回的是*User
func (r *userRepo) findById(ctx context.Context, id uint32) (*User, error) {
	// 先查看缓存有无对应的user信息
	u, err := r.getCachedUserById(ctx, id)
	if err == nil {
		return u, nil
	}
	// 如果遇到错误，但不是key不存在的错误，则输出到日志里，继续查询数据库
	if !errors.Is(err, redis.Nil) {
		r.log.Errorf("get user cache by id %d failed: %s", id, err.Error())
	}
	u = new(User)
	err = r.db.WithContext(ctx).Model(u).
		Where("id = ?", id).
		First(u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	// 缓存用户信息
	go func() {
		err := r.cacheUserById(u)
		if err != nil {
			r.log.Errorf("cache user by id %d failed: %s", id, err.Error())
		}
	}()
	return u, nil
}

// FindByIds .
func (r *userRepo) FindByIds(ctx context.Context, userId uint32, ids []uint32) ([]*biz.User, error) {
	result := make([]*biz.User, 0, len(ids))
	// 记录查询过的id，避免出现查询重复的id
	once := make(map[uint32]int)
	session := r.db.WithContext(ctx)
	for _, id := range ids {
		// 重复id无需查询，从已查询的结果中获取
		if idx, ok := once[id]; ok {
			result = append(result, result[idx])
			continue
		}
		// 先查看缓存有无对应的user信息
		u, err := r.getCachedUserById(ctx, id)
		if err == nil {
			user := new(biz.User)
			copier.Copy(user, u)
			result = append(result, user)
			once[id] = len(result) - 1
			continue
		}
		// 如果遇到错误，但不是key不存在的错误，则输出到日志里，继续查询数据库
		if !errors.Is(err, redis.Nil) {
			r.log.Errorf("get user cache by id %d failed: %s", id, err.Error())
		}
		u = new(User)
		// 对于唯一id，进行查询
		err = session.Model(u).
			Where("id = ?", id).
			First(u).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		// 缓存用户信息
		go func(id uint32) {
			err := r.cacheUserById(u)
			if err != nil {
				r.log.Errorf("cache user by id %d failed: %s", id, err.Error())
			}
		}(u.Id)
		user := new(biz.User)
		copier.Copy(user, u)
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

func (r *userRepo) findByUsername(ctx context.Context, username string) (*User, error) {
	// 先查看缓存有无对应的user信息
	u, err := r.getCachedUserByUsername(ctx, username)
	if err == nil {
		return u, nil
	}
	// 如果遇到错误，但不是key不存在的错误，则输出到日志里，继续查询数据库
	if !errors.Is(err, redis.Nil) {
		r.log.Errorf("get user cache by username %s failed: %s", username, err.Error())
	}
	u = new(User)
	err = r.db.WithContext(ctx).Model(u).
		Where("username = ?", username).
		First(u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	// 缓存用户信息
	go func() {
		err = r.cacheUserByUsername(u)
		if err != nil {
			r.log.Errorf("cache user by username %s failed: %s", u.Username, err.Error())
		}
	}()
	return u, nil
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

// UpdateFollow .
func (r *userRepo) UpdateFollow(ctx context.Context, id uint32, followChange int32) error {
	user, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(user.FollowCount, followChange)
	return r.db.WithContext(ctx).Model(user).
		Update("follow_count", newValue).Error
}

// UpdateFollower .
func (r *userRepo) UpdateFollower(ctx context.Context, id uint32, followerChange int32) error {
	user, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(user.FollowerCount, followerChange)
	return r.db.WithContext(ctx).Model(user).
		Update("follower_count", newValue).Error
}

// UpdateFavorited .
func (r *userRepo) UpdateFavorited(ctx context.Context, id uint32, favoritedChange int32) error {
	user, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(user.TotalFavorited, favoritedChange)
	return r.db.WithContext(ctx).Model(user).
		Update("total_favorited", newValue).Error
}

// UpdateWork .
func (r *userRepo) UpdateWork(ctx context.Context, id uint32, workChange int32) error {
	user, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(user.WorkCount, workChange)
	return r.db.WithContext(ctx).Model(user).
		Update("work_count", newValue).Error
}

// UpdateFavorite .
func (r *userRepo) UpdateFavorite(ctx context.Context, id uint32, favoriteChange int32) error {
	user, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(user.FavoriteCount, favoriteChange)
	return r.db.WithContext(ctx).Model(user).
		Update("favorite_count", newValue).Error
}

// cacheUserById 用id来缓存User信息
func (r *userRepo) cacheUserById(user *User) error {
	bs, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return r.rdb.Set(context.TODO(), getUserCachedKeyById(user.Id), bs, time.Duration(FixedCacheExpire)*time.Minute).Err()
}

// cacheUserById 用username来缓存User信息
func (r *userRepo) cacheUserByUsername(user *User) error {
	bs, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return r.rdb.Set(context.TODO(), getUserCachedKeyByUsername(user.Username), bs, time.Duration(FixedCacheExpire)*time.Minute).Err()
}

// getCachedUserById 根据id来获取缓存的User信息
// error 可能是key不存在
func (r *userRepo) getCachedUserById(ctx context.Context, id uint32) (*User, error) {
	user := new(User)
	bs, err := r.rdb.Get(ctx, getUserCachedKeyById(id)).Bytes()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bs, user)
	return user, err
}

// getCachedUserByUsername 根据username来获取缓存的User信息
// error 可能是key不存在
func (r *userRepo) getCachedUserByUsername(ctx context.Context, username string) (*User, error) {
	user := new(User)
	bs, err := r.rdb.Get(ctx, getUserCachedKeyByUsername(username)).Bytes()
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

// getUserCachedKeyByUsername 获取根据username拼接成用于缓存User的key
func getUserCachedKeyByUsername(username string) string {
	return fmt.Sprintf("user:username:%s", username)
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
