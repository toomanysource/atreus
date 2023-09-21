package data

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	favoritev1 "github.com/toomanysource/atreus/api/favorite/service/v1"
	userv1 "github.com/toomanysource/atreus/api/user/service/v1"

	"github.com/toomanysource/atreus/middleware"

	"github.com/segmentio/kafka-go"

	"github.com/toomanysource/atreus/pkg/kafkaX"

	"github.com/toomanysource/atreus/app/publish/service/internal/biz"
	"github.com/toomanysource/atreus/pkg/ffmpegX"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/minio/minio-go/v7"
)

var ErrVideoMissing = errors.New("video missing")

const (
	VideoCount  = 30
	FrameNumber = 60
)

type Video struct {
	Id            uint32 `gorm:"column:id;primary_key;auto_increment"`
	AuthorID      uint32 `gorm:"column:author_id;not null;index:idx_author_id"`
	Title         string `gorm:"column:title;not null;size:255"`
	PlayUrl       string `gorm:"column:play_url;not null"`
	CoverUrl      string `gorm:"column:cover_url;not null"`
	FavoriteCount uint32 `gorm:"column:favorite_count;not null;default:0"`
	CommentCount  uint32 `gorm:"column:comment_count;not null;default:0"`
	CreatedAt     int64  `gorm:"column:created_at"`
}

type UserRepo interface {
	GetUserInfos(context.Context, uint32, []uint32) ([]*biz.User, error)
}

type FavoriteRepo interface {
	IsFavorite(context.Context, uint32, []uint32) ([]bool, error)
}

type publishRepo struct {
	data         *Data
	kfk          KfkReader
	favoriteRepo FavoriteRepo
	userRepo     UserRepo
	log          *log.Helper
}

func NewPublishRepo(
	data *Data, userConn userv1.UserServiceClient, favoriteConn favoritev1.FavoriteServiceClient, logger log.Logger,
) biz.PublishRepo {
	return &publishRepo{
		data:         data,
		kfk:          data.kfkReader,
		favoriteRepo: NewFavoriteRepo(favoriteConn),
		userRepo:     NewUserRepo(userConn),
		log:          log.NewHelper(log.With(logger, "module", "data/publish")),
	}
}

// UploadAll 上传视频和封面
func (r *publishRepo) UploadAll(ctx context.Context, fileBytes []byte, title string) error {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	var wg sync.WaitGroup
	errChan := make(chan error)
	// 生成封面
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := r.UploadCoverImage(context.Background(), fileBytes, title)
		if err != nil {
			errChan <- err
			return
		}
	}()
	wg.Add(1)
	// 上传视频
	go func() {
		defer wg.Done()
		err := r.UploadVideo(context.Background(), fileBytes, title)
		if err != nil {
			errChan <- err
			return
		}
	}()
	wg.Wait()
	select {
	case err := <-errChan:
		return err
	default:
		go func() {
			// 获取视频和封面的url
			err := r.SaveVideoInfo(context.Background(), title, userId)
			if err != nil {
				r.log.Error(err)
				return
			}
			err = kafkaX.Update(r.data.kfkWriter, strconv.Itoa(int(userId)), "1")
			if err != nil {
				r.log.Error(err)
				return
			}
		}()
	}
	return nil
}

// GetFeedList 获取视频列表
func (r *publishRepo) GetFeedList(
	ctx context.Context, latestTime string,
) (int64, []*biz.Video, error) {
	userId := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	if latestTime == "0" {
		latestTime = strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	times, err := strconv.Atoi(latestTime)
	if err != nil {
		return 0, nil, err
	}
	videoList, err := r.GetVideoByTime(ctx, int64(times))
	if err != nil {
		return 0, nil, err
	}
	if len(videoList) == 0 {
		return 0, nil, nil
	}
	err = r.UpdateUrl(ctx, videoList)
	if err != nil {
		return 0, nil, err
	}
	nextTime := videoList[len(videoList)-1].CreatedAt
	vl, err := r.GetVideoAuthor(ctx, userId, videoList)
	if err != nil {
		return 0, nil, err
	}

	// userId == 0 代表未登录
	if userId != 0 {
		videoIds := make([]uint32, 0, len(videoList))
		for _, video := range videoList {
			videoIds = append(videoIds, video.Id)
		}
		isFavoriteList, err := r.favoriteRepo.IsFavorite(ctx, userId, videoIds)
		if err != nil {
			return 0, nil, err
		}
		for i, video := range vl {
			video.IsFavorite = isFavoriteList[i]
		}
		return nextTime, vl, err
	}
	for _, video := range vl {
		video.IsFavorite = false
	}
	return nextTime, vl, err
}

// GetVideosByUserId 根据用户id获取视频列表
func (r *publishRepo) GetVideosByUserId(ctx context.Context, userId uint32) ([]*biz.Video, error) {
	var videoList []*Video
	userID := ctx.Value(middleware.UserIdKey("user_id")).(uint32)
	err := r.data.db.WithContext(ctx).Where("author_id = ?", userId).Find(&videoList).Error
	if err != nil {
		return nil, errors.Join(ErrMysqlQuery, err)
	}
	if len(videoList) == 0 {
		return nil, nil
	}
	err = r.UpdateUrl(ctx, videoList)
	if err != nil {
		return nil, err
	}
	users, err := r.userRepo.GetUserInfos(ctx, userID, []uint32{userId})
	if err != nil {
		return nil, err
	}
	videoIds := make([]uint32, 0, len(videoList))
	for _, video := range videoList {
		videoIds = append(videoIds, video.Id)
	}
	isFavoriteList, err := r.favoriteRepo.IsFavorite(ctx, userId, videoIds)
	if err != nil {
		return nil, err
	}
	vl := make([]*biz.Video, 0, len(videoList))
	for i, video := range videoList {
		vl = append(vl, &biz.Video{
			ID:            video.Id,
			Author:        users[0],
			PlayUrl:       video.PlayUrl,
			CoverUrl:      video.CoverUrl,
			FavoriteCount: video.FavoriteCount,
			CommentCount:  video.CommentCount,
			IsFavorite:    isFavoriteList[i],
			Title:         video.Title,
		})
	}
	return vl, nil
}

// GetVideosByVideoIds 根据视频id列表获取视频列表
func (r *publishRepo) GetVideosByVideoIds(ctx context.Context, userId uint32, videoIds []uint32) ([]*biz.Video, error) {
	if len(videoIds) == 0 {
		return nil, nil
	}
	var videoList []*Video
	err := r.data.db.WithContext(ctx).Where("id IN ?", videoIds).Find(&videoList).Error
	if err != nil {
		return nil, errors.Join(ErrMysqlQuery, err)
	}
	if len(videoList) < len(videoIds) {
		return nil, ErrVideoMissing
	}
	err = r.UpdateUrl(ctx, videoList)
	if err != nil {
		return nil, err
	}
	return r.GetVideoAuthor(ctx, userId, videoList)
}

// SaveVideoInfo 保存视频信息
func (r *publishRepo) SaveVideoInfo(ctx context.Context, title string, userId uint32) error {
	playUrl, coverUrl, err := r.GetRemoteVideoInfo(ctx, title)
	if err != nil {
		return err
	}
	v := &Video{
		AuthorID:      userId,
		Title:         title,
		PlayUrl:       playUrl,
		CoverUrl:      coverUrl,
		FavoriteCount: 0,
		CommentCount:  0,
		CreatedAt:     time.Now().UnixMilli(),
	}
	if err = r.data.db.WithContext(ctx).Create(v).Error; err != nil {
		return errors.Join(ErrMysqlInsert, err)
	}
	return nil
}

// GetRemoteVideoInfo 获取远程视频及封面url
func (r *publishRepo) GetRemoteVideoInfo(ctx context.Context, title string) (playURL, coverURL string, err error) {
	hours, days := 24, 7
	urls, err := r.data.oss.GetFileURL(
		ctx, "oss", "videos/"+title+".mp4", time.Hour*time.Duration(hours*days))
	if err != nil {
		return "", "", fmt.Errorf("get remote video info err, %w", err)
	}
	playURL = urls.String()
	urls, err = r.data.oss.GetFileURL(
		ctx, "oss", "images/"+title+".png", time.Hour*time.Duration(hours*days))
	if err != nil {
		return "", "", fmt.Errorf("get remote video info err, %w", err)
	}
	coverURL = urls.String()
	return
}

// UploadVideo 上传视频
func (r *publishRepo) UploadVideo(ctx context.Context, fileBytes []byte, title string) error {
	reader := bytes.NewReader(fileBytes)
	return r.data.oss.UploadSizeFile(
		ctx, "oss", "videos/"+title+".mp4", reader, reader.Size(), minio.PutObjectOptions{
			ContentType: "video/mp4",
		},
	)
}

// UploadCoverImage 上传封面
func (r *publishRepo) UploadCoverImage(ctx context.Context, fileBytes []byte, title string) error {
	coverReader, err := r.GenerateCoverImage(fileBytes)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(coverReader)
	if err != nil {
		return errors.Join(ErrFileRead, err)
	}
	coverBytes := bytes.NewReader(data)
	return r.data.oss.UploadSizeFile(
		ctx, "oss", "images/"+title+".png", coverBytes, coverBytes.Size(), minio.PutObjectOptions{
			ContentType: "image/png",
		},
	)
}

// GenerateCoverImage 生成封面
func (r *publishRepo) GenerateCoverImage(fileBytes []byte) (io.Reader, error) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "tempFile-*")
	if err != nil {
		return nil, errors.Join(ErrFileCreate, err)
	}
	defer os.Remove(tempFile.Name())
	if _, err = tempFile.Write(fileBytes); err != nil {
		return nil, errors.Join(ErrFileWrite, err)
	}
	// 调用ffmpeg 生成封面
	return ffmpegX.ReadFrameAsImage(tempFile.Name(), FrameNumber)
}

// GetVideoByTime 根据时间获取视频列表
func (r *publishRepo) GetVideoByTime(ctx context.Context, times int64) ([]*Video, error) {
	var videoList []*Video
	err := r.data.db.WithContext(ctx).Where("created_at < ?", times).
		Order("created_at desc").Limit(VideoCount).Find(&videoList).Error
	if err != nil {
		return nil, errors.Join(ErrMysqlQuery, err)
	}
	return videoList, nil
}

// GetVideoAuthor 获取视频作者
func (r *publishRepo) GetVideoAuthor(ctx context.Context, userId uint32, videoList []*Video) ([]*biz.Video, error) {
	userIdMap := make(map[uint32]*biz.User)
	// 去重
	for _, video := range videoList {
		if _, ok := userIdMap[video.AuthorID]; !ok {
			userIdMap[video.AuthorID] = &biz.User{}
			continue
		}
	}
	var userIds []uint32
	for k := range userIdMap {
		userIds = append(userIds, k)
	}
	users, err := r.userRepo.GetUserInfos(ctx, userId, userIds)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		userIdMap[user.ID] = user
	}
	vl := make([]*biz.Video, 0, len(videoList))
	for _, video := range videoList {
		vl = append(vl, &biz.Video{
			ID:            video.Id,
			Author:        userIdMap[video.AuthorID],
			PlayUrl:       video.PlayUrl,
			CoverUrl:      video.CoverUrl,
			FavoriteCount: video.FavoriteCount,
			CommentCount:  video.CommentCount,
			IsFavorite:    false,
			Title:         video.Title,
		})
	}
	return vl, nil
}

// UpdateFavoriteCount 更新点赞数
func (r *publishRepo) UpdateFavoriteCount(ctx context.Context, videoId uint32, favoriteChange int32) error {
	var video Video
	err := r.data.db.WithContext(ctx).Where("id = ?", videoId).First(&video).Error
	if err != nil {
		return errors.Join(ErrMysqlQuery, err)
	}
	newCount := calculate(video.FavoriteCount, favoriteChange)
	err = r.data.db.WithContext(ctx).Where("id = ?", videoId).Update("favorite_count", newCount).Error
	if err != nil {
		return errors.Join(ErrMysqlUpdate, err)
	}
	return err
}

// UpdateCommentCount 更新评论数
func (r *publishRepo) UpdateCommentCount(ctx context.Context, videoId uint32, change int32) error {
	var video Video
	err := r.data.db.WithContext(ctx).Where("id = ?", videoId).First(&video).Error
	if err != nil {
		return errors.Join(ErrMysqlQuery, err)
	}
	newCount := calculate(video.CommentCount, change)
	err = r.data.db.WithContext(ctx).Where("id = ?", videoId).
		Update("comment_count", newCount).Error
	if err != nil {
		return errors.Join(ErrMysqlUpdate, err)
	}
	return nil
}

// CheckUrl 检查url是否过期
func (r *publishRepo) CheckUrl(accessUrl string) (bool, error) {
	parseUrl, err := url.Parse(accessUrl)
	if err != nil {
		return false, err
	}
	dateStr := parseUrl.Query().Get("X-Amz-Date")
	dateInt, err := time.Parse("20060102T150405Z", dateStr)
	if err != nil {
		return false, err
	}
	// 7天后过期,提前一个小时生成新的url
	hours, days := 24, 7
	now := time.Now().Add(time.Hour).UTC()
	return now.Before(dateInt.Add(time.Hour * time.Duration(hours*days))), nil
}

// UpdateUrl 更新url
func (r *publishRepo) UpdateUrl(ctx context.Context, videoList []*Video) error {
	for _, video := range videoList {
		ok, err := r.CheckUrl(video.PlayUrl)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		video.PlayUrl, video.CoverUrl, err = r.GetRemoteVideoInfo(ctx, video.Title)
		if err != nil {
			return err
		}
		err = r.UpdateDatabaseUrl(ctx, video.Id, video.PlayUrl, video.CoverUrl)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateDatabaseUrl 更新数据库url
func (r *publishRepo) UpdateDatabaseUrl(ctx context.Context, videoId uint32, playUrl, coverUrl string) error {
	err := r.data.db.WithContext(ctx).Where("id = ?", videoId).
		Updates(&Video{PlayUrl: playUrl, CoverUrl: coverUrl}).Error
	if err != nil {
		return errors.Join(ErrMysqlUpdate, err)
	}
	return nil
}

// InitUpdateFavoriteQueue 初始化更新点赞数队列
func (r *publishRepo) InitUpdateFavoriteQueue() {
	kafkaX.Reader(r.kfk.favorite, r.log, func(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
		videoId, err := strconv.Atoi(string(msg.Key))
		if err != nil {
			r.log.Error(ErrKafkaReader, err)
			return
		}
		change, err := strconv.Atoi(string(msg.Value))
		if err != nil {
			r.log.Error(ErrKafkaReader, err)
			return
		}
		err = r.UpdateFavoriteCount(ctx, uint32(videoId), int32(change))
		if err != nil {
			r.log.Error(ErrKafkaReader, err)
			return
		}
	})
}

// InitUpdateCommentQueue 初始化更新评论数队列
func (r *publishRepo) InitUpdateCommentQueue() {
	kafkaX.Reader(r.kfk.comment, r.log, func(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
		videoId, err := strconv.Atoi(string(msg.Key))
		if err != nil {
			r.log.Error(ErrKafkaReader, err)
			return
		}
		change, err := strconv.Atoi(string(msg.Value))
		if err != nil {
			r.log.Error(ErrKafkaReader, err)
			return
		}
		err = r.UpdateCommentCount(ctx, uint32(videoId), int32(change))
		if err != nil {
			r.log.Error(ErrKafkaReader, err)
			return
		}
	})
}

// calculate 转换计算类型
func calculate(src uint32, mod int32) uint32 {
	if mod < 0 {
		mod = -mod
		if src < uint32(mod) {
			return 0
		}
		return src - uint32(mod)
	}
	return src + uint32(mod)
}
