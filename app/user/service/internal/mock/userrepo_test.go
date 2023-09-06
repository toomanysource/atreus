package mock_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toomanysource/atreus/app/user/service/internal/biz"
	"github.com/toomanysource/atreus/app/user/service/internal/mock"
)

var userRepo = mock.NewUserRepo()

func testCreate(t *testing.T) {
	ctx := context.Background()
	tests := []*biz.User{
		{Id: 9, Username: "dajun", Password: "junda", Name: "dajun"},
		{Id: 10, Username: "mimi", Password: "mimi", Name: "mimi"},
		{Id: 11, Username: "noname", Password: "nameno", Name: "noname"},
	}
	rowsBeforeTest := mock.GetUserTableLength()
	for _, tt := range tests {
		_, err := userRepo.Create(ctx, tt)
		assert.NoError(t, err)
	}
	rowsAfterTest := mock.GetUserTableLength()
	assert.Equal(t, rowsBeforeTest+len(tests), rowsAfterTest)
}

func testFindById(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		id       uint32
		username string
	}{
		{1, "xiaoming"},
		{2, "xiaohong"},
		{3, "liuzi"},
		{4, "lengzi"},
		{5, "aniu"},
		{6, "erlengzi"},
	}
	for _, tt := range tests {
		user, err := userRepo.FindById(ctx, tt.id)
		assert.NoError(t, err)
		assert.Equal(t, tt.username, user.Username)
	}
}

func testFindByIds(t *testing.T) {
	ctx := context.Background()
	tests := map[uint32]string{
		1: "xiaoming",
		2: "xiaohong",
		3: "liuzi",
		4: "lengzi",
		5: "aniu",
		6: "erlengzi",
	}
	ids := []uint32{}
	for k := range tests {
		ids = append(ids, k)
	}
	users, err := userRepo.FindByIds(ctx, 0, ids)
	assert.NoError(t, err)
	for i := range users {
		name := tests[users[i].Id]
		assert.Equal(t, name, users[i].Name)
	}
}

func testFindByUsername(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		username string
		id       uint32
	}{
		{"xiaoming", 1},
		{"xiaohong", 2},
		{"liuzi", 3},
		{"lengzi", 4},
		{"aniu", 5},
		{"erlengzi", 6},
	}
	for _, tt := range tests {
		user, err := userRepo.FindByUsername(ctx, tt.username)
		assert.NoError(t, err)
		assert.Equal(t, tt.id, user.Id)
	}
}

func TestUserRepo(t *testing.T) {
	t.Run("TestUserRepoCreateUser", testCreate)
	t.Run("TestUserRepoFindUserById", testFindById)
	t.Run("TestUserRepoFindUsersByIds", testFindByIds)
	t.Run("TestUserRepoFindUserByUsername", testFindByUsername)
}
