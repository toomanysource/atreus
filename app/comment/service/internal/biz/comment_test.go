package biz

import (
	"context"
	"os"
	"testing"

	"github.com/toomanysource/atreus/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

var (
	ctx              = context.Background()
	testCommentsData = map[uint32]*Comment{
		1: {
			Id: 1,
			User: &User{
				Id:   1,
				Name: "hahah",
			},
			Content:    "bushuwu1",
			CreateDate: "08-01",
		},
		2: {
			Id: 2,
			User: &User{
				Id:   1,
				Name: "hahah",
			},
			Content:    "dadawd",
			CreateDate: "08-02",
		},
		3: {
			Id: 3,
			User: &User{
				Id:   2,
				Name: "sefafa",
			},
			Content:    "bdzxvzad",
			CreateDate: "08-03",
		},
		4: {
			Id: 4,
			User: &User{
				Id:   1,
				Name: "hahah",
			},
			Content:    "bvrbr",
			CreateDate: "08-03",
		},
		5: {
			Id: 5,
			User: &User{
				Id:   3,
				Name: "brbs",
			},
			Content:    "bdadawfvrd",
			CreateDate: "08-04",
		},
		6: {
			Id: 6,
			User: &User{
				Id:   5,
				Name: "bgssev",
			},
			Content:    "bdafagaagaga",
			CreateDate: "08-05",
		},
	}
)
var autoCount uint32 = 7

type MockCommentRepo struct{}

func (m *MockCommentRepo) CreateComment(ctx context.Context, videoId uint32, commentText, createTime string) (*Comment, error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	comment := &Comment{
		Id: autoCount,
		User: &User{
			Id:   userId,
			Name: "hahah",
		},
		Content:    commentText,
		CreateDate: "08-01",
	}
	testCommentsData[comment.Id] = comment
	autoCount++
	return comment, nil
}

func (m *MockCommentRepo) DeleteComment(ctx context.Context, videoId, commentId uint32) error {
	delete(testCommentsData, commentId)
	return nil
}

func (m *MockCommentRepo) GetComments(ctx context.Context, videoId uint32) ([]*Comment, error) {
	var comments []*Comment
	for _, comment := range testCommentsData {
		comments = append(comments, comment)
	}
	return comments, nil
}

func (m *MockCommentRepo) GetCommentNumber(ctx context.Context, videoId uint32) (int64, error) {
	return int64(len(testCommentsData)), nil
}

type MockUserRepo struct{}

func (m *MockUserRepo) GetUserInfos(context.Context, uint32, []uint32) ([]*User, error) {
	return []*User{
		{Id: 1, Name: "hvv"},
		{Id: 2, Name: "hvvdad"},
		{Id: 3, Name: "hfaf"},
		{Id: 4, Name: "vaw"},
		{Id: 5, Name: "wadaw"},
		{Id: 6, Name: "wada"},
	}, nil
}

var (
	mockRepo     = &MockCommentRepo{}
	mockUserRepo = &MockUserRepo{}
)

var useCase *CommentUseCase

func TestMain(m *testing.M) {
	ctx = context.WithValue(ctx, middleware.UserIdKey("user_id"), uint32(1))
	useCase = NewCommentUseCase(mockRepo, mockUserRepo, log.DefaultLogger)
	r := m.Run()
	os.Exit(r)
}

func TestCommentUseCase_CommentAction(t *testing.T) {
	_, err := useCase.CommentAction(
		ctx, 1, 0, 1, "test")
	assert.Nil(t, err)
	_, err = useCase.CommentAction(
		ctx, 1, 1, 2, "")
	assert.Nil(t, err)
}

func TestCommentUseCase_GetCommentList(t *testing.T) {
	comments, err := useCase.GetCommentList(ctx, 1)
	assert.Nil(t, err)
	assert.Equal(t, len(comments), len(testCommentsData))
}
