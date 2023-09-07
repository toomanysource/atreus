package mock

import (
	"context"
	"errors"

	"github.com/jinzhu/copier"
	"gorm.io/gorm"

	"github.com/toomanysource/atreus/app/user/service/internal/biz"
)

type UserDetail struct {
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
	DeletedAt       gorm.DeletedAt
}

var userTable = []*UserDetail{
	{1, "xiaoming", "mingxiao", "xiaoming", 1, 1, "avatar_1", "background_image_1", "signature_1", 1, 1, 1, gorm.DeletedAt{}},
	{2, "xiaohong", "hongxiao", "xiaohong", 2, 2, "avatar_2", "background_image_2", "signature_2", 2, 2, 2, gorm.DeletedAt{}},
	{3, "liuzi", "ziliu", "liuzi", 3, 3, "avatar_3", "background_image_3", "signature_3", 3, 3, 3, gorm.DeletedAt{}},
	{4, "lengzi", "zileng", "lengzi", 4, 4, "avatar_4", "background_image_4", "signature_4", 4, 4, 4, gorm.DeletedAt{}},
	{5, "aniu", "niua", "aniu", 5, 5, "avatar_5", "background_image_5", "signature_5", 5, 5, 5, gorm.DeletedAt{}},
	{6, "erlengzi", "zilenger", "erlengzi", 6, 6, "avatar_6", "background_image_6", "signature_6", 6, 6, 6, gorm.DeletedAt{}},
}

type userRepo struct{}

func NewUserRepo() biz.UserRepo {
	return &userRepo{}
}

func (r *userRepo) Create(ctx context.Context, user *biz.User) (*biz.User, error) {
	u := new(UserDetail)
	copier.Copy(u, user)
	for i := range userTable {
		if u.Id == userTable[i].Id {
			return nil, errors.New("duplicated unique key")
		}
	}
	userTable = append(userTable, u)
	return user, nil
}

func (r *userRepo) FindById(ctx context.Context, uid uint32) (*biz.User, error) {
	user := new(biz.User)
	for i := range userTable {
		if uid == userTable[i].Id {
			copier.Copy(user, userTable[i])
			return user, nil
		}
	}
	return nil, errors.New("user not found by this id")
}

func (r *userRepo) FindByIds(ctx context.Context, ids []uint32) ([]*biz.User, error) {
	record := make(map[uint32]struct{}, len(ids))
	for _, id := range ids {
		record[id] = struct{}{}
	}
	users := []*biz.User{}
	for i := range userTable {
		if _, ok := record[userTable[i].Id]; ok {
			user := new(biz.User)
			copier.Copy(user, userTable[i])
			users = append(users, user)
		}
	}
	return users, nil
}

func (r *userRepo) FindKeyInfoByUsername(ctx context.Context, username string) (*biz.User, error) {
	user := new(biz.User)
	for i := range userTable {
		if username == userTable[i].Username {
			copier.Copy(user, userTable[i])
			return user, nil
		}
	}
	return nil, errors.New("user not found by username")
}

func (r *userRepo) RunUpdateFollowListener() {
	// update follow codes
}

func (r *userRepo) RunUpdateFollowerListener() {
	// update follower codes
}

func (r *userRepo) RunUpdateFavoriteListener() {
	// update favored codes
}

func (r *userRepo) RunUpdateFavoredListener() {
	// update publish codes
}

func (r *userRepo) RunUpdateWorkListener() {
	// update favorite codes
}

var _ biz.UserRepo = (*userRepo)(nil)
