package errorX

import "errors"

const (
	CodeSuccess = 0
	CodeFailed  = 300
)

var (
	ErrInValidActionType       = errors.New("invalid action type")
	ErrInternal                = errors.New("internal error")
	ErrCopy                    = errors.New("copy error")
	ErrMysqlDelete             = errors.New("mysql delete error")
	ErrMysqlQuery              = errors.New("mysql query error")
	ErrRedisQuery              = errors.New("redis query error")
	ErrRedisDelete             = errors.New("redis delete error")
	ErrRedisTransaction        = errors.New("redis transaction error")
	ErrJsonMarshal             = errors.New("json marshal error")
	ErrRedisSet                = errors.New("redis set error")
	ErrUserServiceResponse     = errors.New("user service response error")
	ErrKafkaWriter             = errors.New("kafka writer error")
	ErrMysqlInsert             = errors.New("mysql insert error")
	ErrStrconvParse            = errors.New("strconv parse error")
	ErrPublishServiceResponse  = errors.New("publish service response error")
	ErrFavoriteServiceResponse = errors.New("favorite service response error")
	ErrFileCreate              = errors.New("file create error")
	ErrFileWrite               = errors.New("file write error")
	ErrFileRead                = errors.New("file read error")
	ErrMysqlUpdate             = errors.New("mysql update error")
	ErrKafkaReader             = errors.New("kafka reader error")
)
