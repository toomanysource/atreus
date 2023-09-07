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
	ErrUserNotFound = errors.New("无法找到此用户")
	ErrInternal     = errors.New("服务内部错误")
)

// User is a user model.
type User struct {
	Id              uint32
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

// UserRepo 定义user存储的方法集合
type UserRepo interface {
	Create(context.Context, *User) (*User, error)
	FindById(context.Context, uint32) (*User, error)
	FindByIds(context.Context, []uint32) ([]*User, error)
	FindKeyInfoByUsername(context.Context, string) (*User, error)
	InitUpdateFollowQueue()
	InitUpdateFollowerQueue()
	InitUpdateFavoredQueue()
	InitUpdatePublishQueue()
	InitUpdateFavoriteQueue()
}

// RelationRepo 定义向relation服务请求的方法集合
type RelationRepo interface {
	IsFollow(ctx context.Context, userId uint32, toUserId []uint32) ([]bool, error)
}

// UserUsecase 是user的用例
type UserUsecase struct {
	userRepo     UserRepo
	relationRepo RelationRepo
	conf         *conf.JWT
	log          *log.Helper
}

func NewUserUsecase(userRepo UserRepo, relationRepo RelationRepo, conf *conf.JWT, logger log.Logger) *UserUsecase {
	logs := log.NewHelper(log.With(logger, "biz", "user_usecase"))
	uc := &UserUsecase{
		userRepo:     userRepo,
		relationRepo: relationRepo,
		conf:         conf,
		log:          logs,
	}
	uc.updateWorker()
	return uc
}

// Register .
func (uc *UserUsecase) Register(ctx context.Context, username, password string) (*User, error) {
	_, err := uc.userRepo.FindKeyInfoByUsername(ctx, username)
	if err == nil {
		return nil, errors.New("用户名已被注册")
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, ErrInternal
	}

	password = common.GenSaltPassword(username, password)
	// Name与Username相同
	regUser := &User{
		Username: username,
		Password: password,
		Name:     username,
	}
	user, err := uc.userRepo.Create(ctx, regUser)
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
	user, err := uc.userRepo.FindKeyInfoByUsername(ctx, username)
	if errors.Is(err, ErrUserNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, ErrInternal
	}

	password = common.GenSaltPassword(username, password)
	if user.Password != password {
		return nil, errors.New("密码不正确")
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
	user, err := uc.userRepo.FindById(ctx, userId)
	if err != nil {
		return nil, ErrInternal
	}
	if user.Username == "" {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// GetInfos
func (uc *UserUsecase) GetInfos(ctx context.Context, userId uint32, userIds []uint32) ([]*User, error) {
	users, err := uc.userRepo.FindByIds(ctx, userIds)
	if err != nil {
		return nil, ErrInternal
	}
	if len(users) == 0 {
		return []*User{}, nil
	}
	// 添加关注详情
	follows, err := uc.relationRepo.IsFollow(ctx, userId, userIds)
	if err != nil {
		uc.log.Errorf("请求relation服务响应失败，原因: %s", err.Error())
		return nil, ErrInternal
	}
	if len(follows) != len(users) {
		uc.log.Error("来自relation服务的响应不匹配，原因: 响应的关注数组长度与待赋值的用户信息数组长度不一致")
		return nil, ErrInternal
	}
	for i := range users {
		users[i].IsFollow = follows[i]
	}
	return users, nil
}

// updateWorker 执行用户信息更新的监听器
func (uc *UserUsecase) updateWorker() {
	go uc.userRepo.InitUpdateFollowQueue()
	go uc.userRepo.InitUpdateFollowerQueue()
	go uc.userRepo.InitUpdatePublishQueue()
	go uc.userRepo.InitUpdateFavoriteQueue()
	go uc.userRepo.InitUpdateFavoredQueue()
}
