package mock_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toomanysource/atreus/app/user/service/internal/mock"
)

var relationRepo = mock.NewRelationRepo()

func TestRelationRepo_IsFollow(t *testing.T) {
	ctx := context.Background()
	expected := make([]bool, 10)
	for i := range expected {
		expected[i] = true
	}
	userIds := make([]uint32, 10)
	for i := range userIds {
		userIds[i] = uint32(i)
	}
	follows, err := relationRepo.IsFollow(ctx, 0, userIds)
	assert.NoError(t, err)
	assert.Equal(t, expected, follows)
}
