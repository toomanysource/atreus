package errorX

import "errors"

const (
	CodeSuccess = 0
	CodeFailed  = 300
)

var (
	ErrInValidActionType      = errors.New("invalid action type")
	ErrUserNotFound           = errors.New("user not found")
	ErrInternal               = errors.New("internal error")
	ErrRegistered             = errors.New("the username has been registered")
	ErrPassword               = errors.New("incorrect password")
	ErrCopy                   = errors.New("copy error")
	ErrMysqlDelete            = errors.New("mysql delete error")
	ErrMysqlQuery             = errors.New("mysql query error")
	ErrRedisQuery             = errors.New("redis query error")
	ErrRedisDelete            = errors.New("redis delete error")
	ErrRedisTransaction       = errors.New("redis transaction error")
	ErrJsonMarshal            = errors.New("json marshal error")
	ErrRedisSet               = errors.New("redis set error")
	ErrUserServiceResponse    = errors.New("user service response error")
	ErrKafkaWriter            = errors.New("kafka writer error")
	ErrCommentNil             = errors.New("comment text is nil")
	ErrMysqlInsert            = errors.New("mysql insert error")
	ErrStrconvParse           = errors.New("strconv parse error")
	ErrPublishServiceResponse = errors.New("publish service response error")
	ErrVideoMissing           = errors.New("video missing")
)
