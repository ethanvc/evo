package xobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
)

type Payload struct {
	Key string `json:"key"`
	Val any    `json:"val"`
}

type Error struct {
	// DO NOT ACCESS DIRECTLY. here use public field only for marshal/unmarshal
	Code    codes.Code `json:"code"`
	Event   []string   `json:"event,omitempty"`
	Msg     string     `json:"msg,omitempty"`
	Details []Payload  `json:"details,omitempty"`
}

func New(code codes.Code, event string) *Error {
	err := &Error{
		Code: code,
	}
	if event != "" {
		err.Event = []string{event}
	}
	return err
}

func (e *Error) GetCode() codes.Code {
	if e == nil {
		return codes.OK
	}
	return e.Code
}

func (e *Error) GetEvent() []string {
	if e == nil {
		return []string{}
	}
	return e.Event
}

func (e *Error) GetMsg() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

func (e *Error) GetDetails() []Payload {
	if e == nil {
		return nil
	}
	return e.Details
}

func (e *Error) SetMsg(msg string) *Error {
	e.Msg = msg
	return e
}

func (e *Error) SetMsgf(format string, args ...any) *Error {
	e.Msg = fmt.Sprintf(format, args...)
	return e
}

func (e *Error) AppendEvent(event string) *Error {
	const maxAllowedEvent = 100
	if len(e.Event) > maxAllowedEvent {
		return e
	}
	e.Event = append(e.Event, event)
	return e
}

func (e *Error) AppendKvEvent(k string, v any) *Error {
	buf := fmt.Sprintf("%s:%v", k, v)
	return e.AppendEvent(buf)
}

const delimiter = ';'

func (e *Error) GetReportEvent() string {
	if e.GetCode() == codes.OK {
		return codes.OK.String()
	}
	buf := bytes.NewBuffer(nil)
	buf.WriteString(e.Code.String())
	for _, event := range e.Event {
		buf.WriteByte(delimiter)
		buf.WriteString(event)
	}
	return buf.String()
}

func (e *Error) Error() string {
	if e.GetCode() == codes.OK {
		return codes.OK.String()
	}
	return e.GetReportEvent() + ";" + e.Msg
}

func (e *Error) LogReport(ctx context.Context, args ...any) *Error {
	return e
}

func (e *Error) clone() *Error {
	newErr := New(e.Code, "").SetMsg(e.Msg)
	newErr.Event = e.Event[:len(e.Event):len(e.Event)]
	newErr.Details = e.Details[:len(e.Details):len(e.Details)]
	return newErr
}

func Code(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	var realErr *Error
	if errors.As(err, &realErr) {
		return realErr.GetCode()
	}
	return codes.Unknown
}

func BlockBusinessErr(err error) error {
	realErr, ok := err.(*Error)
	if !ok {
		return err
	}
	if realErr == nil {
		return nil
	}
	switch realErr.GetCode() {
	case codes.Unknown, codes.Internal, codes.DeadlineExceeded, codes.Aborted,
		codes.Unimplemented, codes.Unavailable, codes.DataLoss:
		return err
	default:
		newErr := realErr.clone()
		newErr.AppendKvEvent("BlockedCode", realErr.GetCode().String())
		newErr.Code = codes.Internal
		return newErr
	}
}

func Convert(err error) *Error {
	if err == nil {
		return New(codes.OK, "")
	}
	realErr, ok := err.(*Error)
	if ok {
		return realErr
	}
	return New(codes.Unknown, "UnknownErr").SetMsg(err.Error())
}

func MakeKvEventStr(kvs ...any) string {
	var buf bytes.Buffer
	if len(kvs)%2 == 1 {
		buf.WriteString("KVEventNotPaired")
	}
	for i := 0; i < len(kvs)-1; i += 2 {
		if buf.Len() > 0 {
			buf.WriteByte(delimiter)
		}
		_, _ = fmt.Fprintf(&buf, "%s:%v", kvs[i], kvs[i+1])
	}
	return buf.String()
}
