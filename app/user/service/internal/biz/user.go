package biz

import (
	"context"
	"errors"
	"time"

	"github.com/toomanysource/atreus/app/user/service/internal/conf"
	"github.com/toomanysource/atreus/pkg/common"

	"github.com/go-kratos/kratos/v2/log"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrInternal     = errors.New("internal error")
)

// User is a user model.
type User struct {
	Id              uint32 `copier:"UserId"`
	Username        string
	Password        string
	Name            string
	FollowCount     uint32
	FollowerCount   uint32
	Avatar          string
	BackgroundImage string
	Signature       string
	TotalFavorited  uint32
	WorkCount       uint32
	FavoriteCount   uint32
	IsFollow        bool
	Token           string
}

// UserRepo is a user repo.
type UserRepo interface {
	Create(context.Context, *User) (*User, error)
	FindById(context.Context, uint32) (*User, error)
	FindByIds(context.Context, uint32, []uint32) ([]*User, error)
	FindByUsername(context.Context, string) (*User, error)
	InitUpdateFollowQueue()
	InitUpdateFollowerQueue()
	InitUpdateFavoredQueue()
	InitUpdatePublishQueue()
	InitUpdateFavoriteQueue()
}

// UserUsecase is a user usecase.
type UserUsecase struct {
	repo UserRepo
	conf *conf.JWT
	log  *log.Helper
}

// NewUserUsecase new a user usecase.
func NewUserUsecase(repo UserRepo, conf *conf.JWT, logger log.Logger) *UserUsecase {
	go repo.InitUpdateFavoredQueue()
	go repo.InitUpdateFollowQueue()
	go repo.InitUpdateFollowerQueue()
	go repo.InitUpdatePublishQueue()
	go repo.InitUpdateFavoriteQueue()
	return &UserUsecase{repo: repo, conf: conf, log: log.NewHelper(logger)}
}

// Register .
func (uc *UserUsecase) Register(ctx context.Context, username, password string) (*User, error) {
	_, err := uc.repo.FindByUsername(ctx, username)
	if err == nil {
		return nil, errors.New("the username has been registered")
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, ErrInternal
	}

	password = common.GenSaltPassword(username, password)

	regUser := &User{
		Username: username,
		Password: password,
		// Name与Username相同
		Name: username,
	}
	user, err := uc.repo.Create(ctx, regUser)
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
	if errors.Is(err, ErrUserNotFound) {
		return nil, errors.New("can not find registered user")
	}
	if err != nil {
		return nil, ErrInternal
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
