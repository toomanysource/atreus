package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/toomanysource/atreus/pkg/common"

	"github.com/toomanysource/atreus/app/message/service/internal/biz"

	"github.com/go-redis/redis/v8"
	"github.com/jinzhu/copier"
	"github.com/segmentio/kafka-go"

	"github.com/go-kratos/kratos/v2/log"
)

type Message struct {
	UId        uint64 `gorm:"column:uid;not null;default:0"`
	FromUserId uint32 `gorm:"column:from_user_id;not null"`
	ToUserId   uint32 `gorm:"column:to_user_id;not null"`
	Content    string `gorm:"column:content;not null"`
	CreateTime int64  `gorm:"column:created_at"`
}

func (Message) TableName() string {
	return "message"
}

type messageRepo struct {
	data *Data
	log  *log.Helper
}

func NewMessageRepo(data *Data, logger log.Logger) biz.MessageRepo {
	return &messageRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *messageRepo) GetMessageList(ctx context.Context, toUserId uint32, preMsgTime int64) ([]*biz.Message, error) {
	// 先在redis缓存中查询是否存在聊天记录列表
	userId := ctx.Value("user_id").(uint32)
	key := setKey(userId, toUserId)
	ok, msgList, err := r.CheckCacheExist(ctx, key, preMsgTime)
	if err != nil {
		return nil, err
	}
	if ok {
		return msgList, nil
	}
	// 加锁防止私聊两用户同时请求导致重复创建
	factor := 20
	ok, err = r.data.cache.SetNX(ctx, "mutex", "", time.Second*time.Duration(factor)).Result()
	if err != nil {
		return nil, fmt.Errorf("redis set mutex error %w", err)
	}
	if ok {
		ok, msgList, err = r.CheckCacheExist(ctx, key, preMsgTime)
		if err != nil {
			return nil, err
		}
		if ok {
			return msgList, nil
		}
		cl, err := r.SearchMessage(ctx, userId, toUserId, preMsgTime)
		if err != nil {
			return nil, err
		}
		// 没有列表则不创建
		if len(cl) == 0 {
			return nil, nil
		}
		go func() {
			if err = r.CacheCreateMessageTransaction(ctx, cl, key); err != nil {
				r.log.Errorf("redis transaction error %w", err)
				return
			}
			r.data.cache.Del(ctx, "mutex")
			r.log.Info("redis transaction success")
		}()
		return cl, nil
	}
	cl, err := r.SearchMessage(ctx, userId, toUserId, preMsgTime)
	if err != nil {
		return nil, err
	}
	return cl, nil
}

func (r *messageRepo) CheckCacheExist(ctx context.Context, key string, preMsgTime int64) (bool, []*biz.Message, error) {
	// 先在redis缓存中查询是否存在聊天记录列表
	count, err := r.data.cache.Exists(ctx, key).Result()
	if err != nil {
		return false, nil, fmt.Errorf("redis query error %w", err)
	}
	if count > 0 {
		msgList, err := r.data.cache.ZRangeByScore(ctx, key, &redis.ZRangeBy{
			Min: fmt.Sprintf("(%f", float64(preMsgTime)),
			Max: "+inf",
		}).Result()
		if err != nil {
			return false, nil, fmt.Errorf("redis query error %w", err)
		}
		if len(msgList) > 0 {
			cl := make([]*biz.Message, 0, len(msgList))
			// 如果存在则直接返回
			for _, v := range msgList {
				co := &biz.Message{}
				if err = json.Unmarshal([]byte(v), co); err != nil {
					return false, nil, fmt.Errorf("json unmarshal error %w", err)
				}
				cl = append(cl, co)
			}
			return true, cl, nil
		}
		return true, nil, nil
	}

	return false, nil, nil
}

// PublishMessage 发送消息
func (r *messageRepo) PublishMessage(ctx context.Context, toUserId uint32, content string) error {
	userId := ctx.Value("user_id").(uint32)
	if userId == toUserId {
		return errors.New("can't send message to yourself")
	}
	createTime := time.Now().UnixMilli()
	// 生成消息uid,解决kafka发送数据库不及时，导致查询时没有数据的问题
	uid := common.NewUUIDInt()
	err := r.MessageProducer(uid, userId, toUserId, content, createTime)
	if err != nil {
		return fmt.Errorf("message producer error, err: %w", err)
	}
	go func() {
		ctx = context.Background()
		key := setKey(userId, toUserId)
		ml := &Message{
			UId:        uid,
			FromUserId: userId,
			ToUserId:   toUserId,
			Content:    content,
			CreateTime: createTime,
		}
		data, err := json.Marshal(ml)
		if err != nil {
			r.log.Errorf("json marshal error %w", err)
			return
		}
		if err = r.data.cache.ZAdd(ctx, key, &redis.Z{
			Score:  float64(createTime),
			Member: string(data),
		}).Err(); err != nil {
			r.log.Errorf("redis store error %w", err)
			return
		}
		r.log.Info("redis store success")
	}()
	return nil
}

// SearchMessage 数据库根据最新消息时间查询消息
func (r *messageRepo) SearchMessage(ctx context.Context, userId, toUserId uint32, preMsgTime int64) (ml []*biz.Message, err error) {
	var mel []*Message
	err = r.data.db.WithContext(ctx).Where(
		"(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
		userId, toUserId, toUserId, userId).Where("created_at >= ?", preMsgTime).
		Order("created_at").Find(&mel).Error
	if err != nil {
		return nil, fmt.Errorf("search message error, err: %w", err)
	}
	if err = copier.Copy(&ml, &mel); err != nil {
		return nil, fmt.Errorf("copy error, err: %w", err)
	}
	return
}

// MessageProducer 生产消息
func (r *messageRepo) MessageProducer(uid uint64, userId, toUserId uint32, content string, time int64) error {
	mg := &Message{
		UId:        uid,
		FromUserId: userId,
		ToUserId:   toUserId,
		Content:    content,
		CreateTime: time,
	}
	byteValue, err := json.Marshal(mg)
	if err != nil {
		return fmt.Errorf("json marshal error, err: %w", err)
	}
	err = r.data.kfk.writer.WriteMessages(context.Background(), kafka.Message{
		Partition: 0,
		Value:     byteValue,
	})
	if err != nil {
		return fmt.Errorf("write message error, err: %w", err)
	}
	return nil
}

// InitStoreMessageQueue 初始化聊天记录存储队列
func (r *messageRepo) InitStoreMessageQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 监听Ctrl+C退出信号
	signChan := make(chan os.Signal, 1)
	signal.Notify(signChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signChan
		cancel()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := r.data.kfk.reader.ReadMessage(ctx)
			if errors.Is(err, context.Canceled) {
				return
			}
			if err != nil {
				r.log.Errorf("read message error, err: %v", err)
			}
			value := msg.Value
			var mg *Message
			err = json.Unmarshal(value, &mg)
			if err != nil {
				r.log.Errorf("json unmarshal error, err: %v", err)
				return
			}
			err = r.InsertMessage(mg.UId, mg.FromUserId, mg.ToUserId, mg.Content)
			if err != nil {
				r.log.Errorf("insert message error, err: %v", err)
				return
			}
			err = r.data.kfk.reader.CommitMessages(ctx, msg)
			if err != nil {
				r.log.Errorf("commit message error, err: %v", err)
				return
			}
			r.log.Infof("message: UserId-%v to UserId-%v: %v ", mg.FromUserId, mg.ToUserId, mg.Content)
		}
	}
}

// InsertMessage 数据库插入消息
func (r *messageRepo) InsertMessage(uid uint64, userId uint32, toUserId uint32, content string) error {
	err := r.data.db.Create(&Message{
		UId:        uid,
		FromUserId: userId,
		ToUserId:   toUserId,
		Content:    content,
		CreateTime: time.Now().UnixMilli(),
	}).Error
	if err != nil {
		return fmt.Errorf("insert message error, err: %w", err)
	}
	return nil
}

// CacheCreateMessageTransaction 缓存创建事务
func (r *messageRepo) CacheCreateMessageTransaction(ctx context.Context, ml []*biz.Message, key string) error {
	// 使用事务将列表存入redis缓存
	_, err := r.data.cache.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		insertList := make([]*redis.Z, 0, len(ml))
		for _, u := range ml {
			data, err := json.Marshal(u)
			if err != nil {
				return fmt.Errorf("json marshal error, err: %w", err)
			}
			insertList = append(insertList, &redis.Z{
				Score:  float64(u.CreateTime),
				Member: string(data),
			})
		}
		if err := pipe.ZAdd(ctx, key, insertList...).Err(); err != nil {
			return fmt.Errorf("redis store error, err : %w", err)
		}
		// 将评论数量存入redis缓存,使用随机过期时间防止缓存雪崩
		begin, end := 360, 720
		err := pipe.Expire(ctx, key, randomTime(time.Minute, begin, end)).Err()
		if err != nil {
			return fmt.Errorf("redis expire error, err : %w", err)
		}
		return nil
	})
	return err
}

// randomTime 随机生成时间
func randomTime(timeType time.Duration, begin, end int) time.Duration {
	return timeType * time.Duration(rand.Intn(end-begin+1)+begin)
}

func setKey(userId, toUserId uint32) string {
	if userId > toUserId {
		userId, toUserId = toUserId, userId
	}
	return fmt.Sprint(strconv.Itoa(int(userId)), "-", strconv.Itoa(int(toUserId)))
}
