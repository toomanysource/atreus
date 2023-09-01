package server

import (
	"bytes"
	"io"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware/validate"

	"github.com/toomanysource/atreus/middleware"
	"github.com/toomanysource/atreus/pkg/errorX"

	"github.com/golang-jwt/jwt/v4"

	v1 "github.com/toomanysource/atreus/api/publish/service/v1"
	"github.com/toomanysource/atreus/app/publish/service/internal/conf"
	"github.com/toomanysource/atreus/app/publish/service/internal/service"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new a user service HTTP server.
func NewHTTPServer(c *conf.Server, t *conf.JWT, publish *service.PublishService, logger log.Logger) *http.Server {
	opts := []http.ServerOption{
		http.ErrorEncoder(errorX.ErrorEncoder),
		http.RequestDecoder(MultipartFormDataDecoder),
		http.Middleware(
			validate.Validator(),
			middleware.TokenParseAll(func(token *jwt.Token) (interface{}, error) {
				return []byte(t.Http.TokenKey), nil
			}),
			recovery.Recovery(),
			logging.Server(log.NewFilter(logger,
				log.FilterKey("args")),
			),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterPublishServiceHTTPServer(srv, publish)
	return srv
}

func MultipartFormDataDecoder(r *http.Request, v interface{}) error {
	// 从Request Header的Content-Type中提取出对应的解码器
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		paRequest := v.(*v1.PublishActionRequest)
		var maxMemory int64 = 32 << 20
		err := r.ParseMultipartForm(maxMemory)
		if err != nil {
			return errors.BadRequest("CODEC", err.Error())
		}
		paRequest.Title = r.FormValue("title")
		paRequest.Token = r.FormValue("token")
		file, _, err := r.FormFile("data")
		if err != nil {
			return errors.BadRequest("CODEC", err.Error())
		}
		var buf bytes.Buffer
		_, err = io.Copy(&buf, file)
		if err != nil {
			return errors.BadRequest("CODEC", err.Error())
		}
		paRequest.Data = buf.Bytes()
		log.Info("app upload success")
		return nil
	}
	codec, ok := http.CodecForRequest(r, "Content-Type")
	// 如果找不到对应的解码器此时会报错
	if !ok {
		return errors.BadRequest("CODEC", r.Header.Get("Content-Type"))
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.BadRequest("CODEC", err.Error())
	}
	if err = codec.Unmarshal(data, v); err != nil {
		return errors.BadRequest("CODEC", err.Error())
	}
	return nil
}
