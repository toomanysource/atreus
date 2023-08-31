package biz

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/toomanysource/atreus/pkg/common"

	"github.com/toomanysource/atreus/middleware"

	"github.com/toomanysource/atreus/app/user/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

type MockUserRepo struct{}

func (m *MockUserRepo) Save(ctx context.Context, user *User) (*User, error) {
	if user.Username == "foo" {
		return user, nil
	}
	return &User{}, nil
}

func (m *MockUserRepo) FindById(ctx context.Context, id uint32) (*User, error) {
	if id == 9 {
		return &User{}, errors.New("error")
	}
	if id < 3 {
		return &User{Id: id}, nil
	}
	s := strconv.Itoa(int(id))
	return &User{Id: id, Username: s, Password: s}, nil
}

func (m *MockUserRepo) FindByIds(ctx context.Context, userId uint32, ids []uint32) ([]*User, error) {
	if userId == 9 {
		return []*User{}, nil
	}
	if userId == 3 {
		return nil, errors.New("error")
	}
	var us []*User
	for _, id := range ids {
		u, _ := m.FindById(context.Background(), id)
		if u.Username == "" {
			continue
		}
		us = append(us, u)
	}
	return us, nil
}

func (m *MockUserRepo) FindByUsername(ctx context.Context, username string) (*User, error) {
	if username == "foo" {
		return &User{}, ErrUserNotFound
	}
	if username == "vvv" {
		return &User{}, ErrInternal
	}
	if username == "xx" {
		password := common.GenSaltPassword(username, username)
		return &User{Username: username, Password: password}, nil
	}
	return &User{Username: username, Password: username}, nil
}

func (m *MockUserRepo) InitUpdateFollowQueue()   {}
func (m *MockUserRepo) InitUpdateFollowerQueue() {}
func (m *MockUserRepo) InitUpdateFavoredQueue()  {}
func (m *MockUserRepo) InitUpdatePublishQueue()  {}
func (m *MockUserRepo) InitUpdateFavoriteQueue() {}

func (m *MockUserRepo) Create(ctx context.Context, user *User) (*User, error) {
	return user, nil
}

var testConfig = &conf.JWT{
	Http: &conf.JWT_Http{
		TokenKey: "AtReUs",
	},
}
var mockRepo = &MockUserRepo{}

var usecase *UserUsecase

func TestMain(m *testing.M) {
	ctx = context.WithValue(ctx, middleware.UserIdKey("userId"), uint32(1))
	usecase = NewUserUsecase(mockRepo, testConfig, log.DefaultLogger)
	r := m.Run()
	os.Exit(r)
}

func TestUserRegister(t *testing.T) {
	// user has been registered
	_, err := usecase.Register(ctx, "xxx", "xxx")
	assert.Error(t, err)
	// user can register
	user, err := usecase.Register(ctx, "foo", "bar")
	assert.NoError(t, err)
	assert.Equal(t, user.Username, "foo")
	_, err = usecase.Register(ctx, "vvv", "vvv")
	assert.Error(t, ErrInternal)
}

func TestUserLogin(t *testing.T) {
	// user not register
	_, err := usecase.Login(ctx, "foo", "bar")
	assert.Error(t, err)
	// incorrect password
	_, err = usecase.Login(ctx, "bar", "foo")
	assert.Error(t, err)
	// login success
	user, err := usecase.Login(ctx, "xx", "xx")
	assert.NoError(t, err)
	assert.Equal(t, user.Username, "xx")
	_, err = usecase.Login(ctx, "vvv", "vvv")
	assert.Error(t, ErrInternal)
}

func TestGetInfo(t *testing.T) {
	// user not found
	_, err := usecase.GetInfo(ctx, 2)
	assert.Error(t, err)
	// user can find
	user, err := usecase.GetInfo(ctx, 4)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, "4")
	_, err = usecase.GetInfo(ctx, 9)
	assert.Error(t, err)
	assert.Equal(t, err, ErrInternal)
}

func TestGetInfos(t *testing.T) {
	// all ids can find user
	ids := []uint32{3, 4, 5, 6, 7}
	userId := uint32(1)
	users, _ := usecase.GetInfos(ctx, userId, ids)
	assert.Equal(t, len(users), len(ids))
	// some ids can not find user
	ids = []uint32{2, 3, 4, 5, 6}
	users, _ = usecase.GetInfos(ctx, userId, ids)
	assert.Equal(t, len(users), len(ids)-1)
	users, err := usecase.GetInfos(ctx, 9, ids)
	assert.Nil(t, err)
	assert.Equal(t, len(users), 0)
	_, err = usecase.GetInfos(ctx, 3, ids)
	assert.Error(t, err)
	assert.Equal(t, err, ErrInternal)
}
