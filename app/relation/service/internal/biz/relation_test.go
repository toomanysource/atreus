package biz

import (
	"context"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"

	"github.com/toomanysource/atreus/middleware"
)

type Followers struct {
	Id         uint32
	FollowerId uint32
}

var (
	ctx      = context.Background()
	useCase  *RelationUseCase
	mock     = &MockRelationRepo{}
	testUser = []*Followers{
		{
			Id:         1,
			FollowerId: 2,
		},
		{
			Id:         2,
			FollowerId: 1,
		},
		{
			Id:         1,
			FollowerId: 3,
		},
		{
			Id:         1,
			FollowerId: 5,
		},
	}
)

type MockRelationRepo struct{}

func (m *MockRelationRepo) GetFollowList(ctx context.Context, userId uint32) (u []*User, err error) {
	for _, v := range testUser {
		if v.FollowerId == userId {
			u = append(u, &User{
				Id: v.Id,
			})
		}
	}
	return
}

func (m *MockRelationRepo) GetFollowerList(ctx context.Context, userId uint32) (u []*User, err error) {
	for _, v := range testUser {
		if v.Id == userId {
			u = append(u, &User{
				Id: v.FollowerId,
			})
		}
	}
	return
}

func (m *MockRelationRepo) Follow(ctx context.Context, userId uint32) error {
	return nil
}

func (m *MockRelationRepo) UnFollow(ctx context.Context, userId uint32) error {
	return nil
}

func (m *MockRelationRepo) IsFollow(ctx context.Context, userId uint32, toUserId []uint32) ([]bool, error) {
	return []bool{true}, nil
}

func TestMain(m *testing.M) {
	ctx = context.WithValue(ctx, middleware.UserIdKey("user_id"), uint32(1))
	useCase = NewRelationUseCase(mock, log.DefaultLogger)
	r := m.Run()
	os.Exit(r)
}

func TestRelationService_GetFollowList(t *testing.T) {
	users, err := useCase.GetFollowList(ctx, 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	users, err = useCase.GetFollowList(ctx, 0)
	assert.Nil(t, err)
}

func TestRelationService_GetFollowerList(t *testing.T) {
	users, err := useCase.GetFollowerList(ctx, 1)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(users))
	users, err = useCase.GetFollowerList(ctx, 0)
	assert.Nil(t, err)
}

func TestRelationService_Action(t *testing.T) {
	err := useCase.Action(ctx, 8, FollowType)
	assert.Nil(t, err)
	err = useCase.Action(ctx, 8, UnfollowType)
	assert.Nil(t, err)
}

func TestRelationService_IsFollow(t *testing.T) {
	b, err := useCase.IsFollow(ctx, 1, []uint32{2})
	assert.Nil(t, err)
	assert.Equal(t, true, b[0])
}
