package obs

import (
	"context"
	"errors"
	"net"
	"reflect"
)

func GetLogErrorObject(err error) *Error {
	if err == nil {
		return nil
	}
	if realErr, ok := errors.AsType[*Error](err); ok {
		return realErr
	}
	if errors.Is(err, context.Canceled) {
		return New(Canceled, "ContextCanceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return New(DeadlineExceeded, "ContextDeadlineExceeded")
	}
	typErr := New(Unknown, err.Error())
	return typErr
}

var errTypeMap = map[reflect.Type]func(err error) *Error{}

func RegisterErrorObject[T error](fn func(err error) *Error) {
	typ := reflect.TypeOf((*T)(nil))
	errTypeMap[typ] = fn
}

func init() {
	RegisterErrorObject[**net.OpError](opError)
}

func opError(err error) *Error {
	typErr, _ := errors.AsType[*net.OpError](err)
	if typErr == nil {
		return nil
	}
	return New(Internal, typErr.Err.Error()).SetMsg(err.Error())
}
