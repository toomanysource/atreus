package mock

import (
	"context"

	"github.com/toomanysource/atreus/app/user/service/internal/biz"
)

type relationRepo struct{}

func NewRelationRepo() biz.RelationRepo {
	return &relationRepo{}
}

func (r *relationRepo) IsFollow(ctx context.Context, userId uint32, userIds []uint32) ([]bool, error) {
	result := make([]bool, len(userIds))
	for i := range result {
		result[i] = true
	}
	return result, nil
}
