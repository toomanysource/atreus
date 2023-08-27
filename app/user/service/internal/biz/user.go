package biz

import (
	"context"
	"errors"
	"time"

	"github.com/toomanysource/atreus/app/user/service/internal/conf"
	"github.com/toomanysource/atreus/pkg/common"

	"github.com/go-kratos/kratos/v2/log"
)

var ErrInternal = errors.New("internal error")

// User is a user model.
type User struct {
	Id              uint32 `gorm:"primary_key"`
	Username        string `gorm:"column:username;not null"`
	Password        string `gorm:"column:password;not null"`
	Name            string `gorm:"column:name;not null"`
	FollowCount     uint32 `gorm:"column:follow_count;not null;default:0"`
	FollowerCount   uint32 `gorm:"column:follower_count;not null;default:0"`
	Avatar          string `gorm:"column:avatar_url;not null;default:''"`
	BackgroundImage string `gorm:"column:background_image_url;not null;default:''"`
	Signature       string `gorm:"column:signature;not null;default:''"`
	TotalFavorited  uint32 `gorm:"column:total_favorited;not null;default:0"`
	WorkCount       uint32 `gorm:"column:work_count;not null;default:0"`
	FavoriteCount   uint32 `grom:"column:favorite_count;not null;default:0"`
	IsFollow        bool   `gorm:"-"`
	Token           string `gorm:"-"`
}

// UserRepo is a user repo.
type UserRepo interface {
	Save(context.Context, *User) (*User, error)
	FindById(context.Context, uint32) (*User, error)
	FindByIds(context.Context, uint32, []uint32) ([]*User, error)
	FindByUsername(context.Context, string) (*User, error)
	UpdateFollow(context.Context, uint32, int32) error
	UpdateFollower(context.Context, uint32, int32) error
	UpdateFavorited(context.Context, uint32, int32) error
	UpdateWork(context.Context, uint32, int32) error
	UpdateFavorite(context.Context, uint32, int32) error
}

// UserUsecase is a user usecase.
type UserUsecase struct {
	repo UserRepo
	conf *conf.JWT
	log  *log.Helper
}

// NewUserUsecase new a user usecase.
func NewUserUsecase(repo UserRepo, conf *conf.JWT, logger log.Logger) *UserUsecase {
	return &UserUsecase{repo: repo, conf: conf, log: log.NewHelper(logger)}
}

// Register .
func (uc *UserUsecase) Register(ctx context.Context, username, password string) (*User, error) {
	user, err := uc.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, ErrInternal
	}
	if user.Username != "" {
		return nil, errors.New("the username has been registered")
	}

	password = common.GenSaltPassword(username, password)

	// save user
	regUser := &User{
		Username: username,
		Password: password,
		// Name is same as username
		Name: username,
	}
	user, err = uc.repo.Save(ctx, regUser)
	if err != nil {
		return nil, ErrInternal
	}

	// 生成 token
	token, err := common.ProduceToken(uc.conf.Http.TokenKey, user.Id, 7*24*time.Hour)
	if err != nil {
		return nil, ErrInternal
	}
	user.Token = token
	return user, nil
}

// Login .
func (uc *UserUsecase) Login(ctx context.Context, username, password string) (*User, error) {
	user, err := uc.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, ErrInternal
	}
	if user.Username == "" {
		return nil, errors.New("can not find registered user")
	}
	password = common.GenSaltPassword(username, password)
	if user.Password != password {
		return nil, errors.New("incorrect password")
	}

	// 生成 token
	token, err := common.ProduceToken(uc.conf.Http.TokenKey, user.Id, 7*24*time.Hour)
	if err != nil {
		return nil, ErrInternal
	}
	user.Token = token
	return user, nil
}

// GetInfo .
func (uc *UserUsecase) GetInfo(ctx context.Context, userId uint32) (*User, error) {
	user, err := uc.repo.FindById(ctx, userId)
	if err != nil {
		return nil, ErrInternal
	}
	if user.Username == "" {
		return nil, errors.New("can not find the user")
	}

	return user, nil
}

// GetInfos .
func (uc *UserUsecase) GetInfos(ctx context.Context, userId uint32, userIds []uint32) ([]*User, error) {
	users, err := uc.repo.FindByIds(ctx, userId, userIds)
	if err != nil {
		return nil, ErrInternal
	}
	if len(users) == 0 {
		return []*User{}, nil
	}

	return users, nil
}

// UpdateFollow .
func (uc *UserUsecase) UpdateFollow(ctx context.Context, userId uint32, followChange int32) error {
	err := uc.repo.UpdateFollow(ctx, userId, followChange)
	if err != nil {
		return ErrInternal
	}

	return nil
}

// UpdateFollower .
func (uc *UserUsecase) UpdateFollower(ctx context.Context, userId uint32, followerChange int32) error {
	err := uc.repo.UpdateFollower(ctx, userId, followerChange)
	if err != nil {
		return ErrInternal
	}

	return nil
}

// UpdateFavorited .
func (uc *UserUsecase) UpdateFavorited(ctx context.Context, userId uint32, favoritedChange int32) error {
	err := uc.repo.UpdateFavorited(ctx, userId, favoritedChange)
	if err != nil {
		return ErrInternal
	}

	return nil
}

// UpdateWork .
func (uc *UserUsecase) UpdateWork(ctx context.Context, userId uint32, workChange int32) error {
	err := uc.repo.UpdateWork(ctx, userId, workChange)
	if err != nil {
		return ErrInternal
	}

	return nil
}

// UpdateFavorite .
func (uc *UserUsecase) UpdateFavorite(ctx context.Context, userId uint32, favoriteChange int32) error {
	err := uc.repo.UpdateFavorite(ctx, userId, favoriteChange)
	if err != nil {
		return ErrInternal
	}

	return nil
}
