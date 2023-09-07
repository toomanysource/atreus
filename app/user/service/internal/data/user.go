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

// UserKeyInfo 是用户信息中的关键信息
// 这些关键信息包含 用户id、用户名和密码
type UserKeyInfo struct {
	Id       uint32 `gorm:"primary_key"`
	Username string `gorm:"column:username"`
	Password string `gorm:"column:password"`
}

type userRepo struct {
	db  *gorm.DB
	kfk KfkReader
	rdb *redis.Client
	log *log.Helper
}

var _ biz.UserRepo = (*userRepo)(nil)

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
	// 给user.Id赋值
	user.Id = u.Id
	// 缓存用户信息
	go func() {
		userDetail := new(UserDetail)
		copier.Copy(userDetail, u)
		r.cacheDetailWithHandleError(userDetail)
	}()
	return user, nil
}

// FindById 返回的是*biz.User
func (r *userRepo) FindById(ctx context.Context, id uint32) (*biz.User, error) {
	userDetail, err := r.findById(ctx, id)
	if err != nil {
		return nil, err
	}
	user := new(biz.User)
	copier.Copy(user, userDetail)
	return user, nil
}

// findById 返回的是*UserDetail
func (r *userRepo) findById(ctx context.Context, id uint32) (*UserDetail, error) {
	// 先查看缓存有无对应的用户信息
	// 如果遇到错误，但不是key不存在的错误，则输出到日志里，继续查询数据库
	userDetail, err := r.getCachedDetailById(ctx, id)
	if err == nil {
		return userDetail, nil
	}
	if !errors.Is(err, redis.Nil) {
		r.log.Errorf("get user cache by id %d failed: %s", id, err.Error())
	}
	// 无法从缓存中获取
	userDetail = new(UserDetail)
	err = r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		First(userDetail).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, biz.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	// 缓存用户信息
	go r.cacheDetailWithHandleError(userDetail)
	return userDetail, nil
}

// FindByIds .
func (r *userRepo) FindByIds(ctx context.Context, ids []uint32) ([]*biz.User, error) {
	result := make([]*biz.User, len(ids))
	// 记录查询过的id，避免出现查询重复的id
	once := make(map[uint32]int)
	session := r.db.WithContext(ctx)
	for i, id := range ids {
		// 重复id无需查询，从已查询的结果中获取
		if idx, ok := once[id]; ok {
			result = append(result, result[idx])
			continue
		}
		// 先查看缓存有无对应的user信息
		userDetail, err := r.getCachedDetailById(ctx, id)
		if err == nil {
			// 添加用户信息
			user := new(biz.User)
			copier.Copy(user, userDetail)
			result[i] = user
			once[id] = i
			continue
		}
		// 如果遇到错误，但不是key不存在的错误，则输出到日志里，继续查询数据库
		if !errors.Is(err, redis.Nil) {
			r.log.Errorf("get user cache by id %d failed: %s", id, err.Error())
		}
		// 对于唯一id，进行查询
		userDetail = new(UserDetail)
		err = session.Model(&User{}).
			Where("id = ?", id).
			First(userDetail).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		// 缓存用户信息
		go r.cacheDetailWithHandleError(userDetail)
		// 添加用户信息
		user := new(biz.User)
		copier.Copy(user, userDetail)
		result[i] = user
		once[id] = i
	}
	return result, nil
}

// FindKeyInfoByUsername .
func (r *userRepo) FindKeyInfoByUsername(ctx context.Context, username string) (*biz.User, error) {
	keyInfo := new(UserKeyInfo)
	err := r.db.WithContext(ctx).Model(&User{}).
		Where("username = ?", username).
		First(keyInfo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, biz.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	user := new(biz.User)
	copier.Copy(user, keyInfo)
	return user, nil
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
	userDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(userDetail.FollowCount, followChange)
	userDetail.FollowCount = newValue
	err = r.cacheDetail(userDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("follow_count", newValue).Error
}

// UpdateFollower .
func (r *userRepo) UpdateFollower(ctx context.Context, id uint32, followerChange int32) error {
	userDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(userDetail.FollowerCount, followerChange)
	userDetail.FollowerCount = newValue
	err = r.cacheDetail(userDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("follower_count", newValue).Error
}

// UpdateFavorited .
func (r *userRepo) UpdateFavorited(ctx context.Context, id uint32, favoritedChange int32) error {
	userDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(userDetail.TotalFavorited, favoritedChange)
	userDetail.TotalFavorited = newValue
	err = r.cacheDetail(userDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("total_favorited", newValue).Error
}

// UpdateWork .
func (r *userRepo) UpdateWork(ctx context.Context, id uint32, workChange int32) error {
	userDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(userDetail.WorkCount, workChange)
	userDetail.WorkCount = newValue
	err = r.cacheDetail(userDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("work_count", newValue).Error
}

// UpdateFavorite .
func (r *userRepo) UpdateFavorite(ctx context.Context, id uint32, favoriteChange int32) error {
	userDetail, err := r.findById(ctx, id)
	if err != nil {
		return err
	}
	newValue := addUint32int32(userDetail.FavoriteCount, favoriteChange)
	userDetail.FavoriteCount = newValue
	err = r.cacheDetail(userDetail)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&User{}).
		Update("favorite_count", newValue).Error
}

// cacheDetail 根据用户信息内的id生成key，并用此key来缓存用户信息
func (r *userRepo) cacheDetail(userDetail *UserDetail) error {
	bs, err := json.Marshal(userDetail)
	if err != nil {
		return err
	}
	return r.rdb.Set(context.TODO(), genCacheKeyById(userDetail.Id), bs, time.Duration(FixedCacheExpire)*time.Minute).Err()
}

// cacheDetailWithHandleError 缓存用户信息并处理错误
// 用于defer函数
func (r *userRepo) cacheDetailWithHandleError(userDetail *UserDetail) {
	err := r.cacheDetail(userDetail)
	if err != nil {
		r.log.Errorf("cache user by id %d failed: %s", userDetail.Id, err.Error())
	}
}

// getCachedDetailById 根据用户id来获取缓存的User信息
// error 可能是key不存在
func (r *userRepo) getCachedDetailById(ctx context.Context, id uint32) (*UserDetail, error) {
	userDetail := new(UserDetail)
	bs, err := r.rdb.Get(ctx, genCacheKeyById(id)).Bytes()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bs, userDetail)
	return userDetail, err
}

// genCacheKeyById 生成根据id拼接成用于缓存用户信息的key
func genCacheKeyById(id uint32) string {
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
