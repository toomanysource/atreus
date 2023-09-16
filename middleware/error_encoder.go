package middleware

import (
	"fmt"
	netHttp "net/http"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport/http"
)

const (
	CodeFailed = 300
)

type Error struct {
	StatusCode int32  `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("error : %v", e.StatusMsg)
}

func New(statusCode int32, statusMsg string) *Error {
	return &Error{StatusCode: statusCode, StatusMsg: statusMsg}
}

func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if se := new(Error); errors.As(err, &se) {
		return New(se.StatusCode, se.StatusMsg)
	}
	if se := new(errors.Error); errors.As(err, &se) {
		return New(se.Code, se.Message)
	}
	return &Error{StatusCode: CodeFailed}
}

func ErrorEncoder(w netHttp.ResponseWriter, r *netHttp.Request, err error) {
	se := FromError(err)
	codec, _ := http.CodecForRequest(r, "Accept")
	body, err := codec.Marshal(se)
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/"+codec.Name())
	_, _ = w.Write(body)
}
