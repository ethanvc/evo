package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"strings"
)

type Status struct {
	s internalStatus
}

func New(code codes.Code, event string) *Status {
	s := &Status{
		s: internalStatus{
			Code:  code,
			Event: event,
		},
	}
	return s
}

func (s *Status) SetCode(code codes.Code) *Status {
	s.s.Code = code
	return s
}

func (s *Status) SetMsg(format string, args ...any) *Status {
	s.s.Msg = fmt.Sprintf(format, args...)
	return s
}

func (s *Status) GetCode() codes.Code {
	if s == nil {
		return codes.OK
	}
	return s.s.Code
}

func (s *Status) GetMsg() string {
	if s == nil {
		return ""
	}
	return s.s.Msg
}

func (s *Status) GetEvent() string {
	if s == nil {
		return ""
	}
	return s.s.Event
}

func (s *Status) Err() error {
	if s.GetCode() == codes.OK {
		return nil
	}
	se := &statusError{
		s: s,
	}
	return se
}

func (s *Status) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return json.Marshal(s.s)
}

func Code(err error) codes.Code {
	s := Convert(err)
	return s.GetCode()
}

func NotFound(err error) bool {
	return Code(err) == codes.NotFound
}

type internalStatus struct {
	Code  codes.Code `json:"code"`
	Msg   string     `json:"msg,omitempty"`
	Event string     `json:"event,omitempty"`
}

type statusError struct {
	s *Status
}

func (se *statusError) Error() string {
	var buf strings.Builder
	buf.WriteString("code=")
	buf.WriteString(se.s.GetCode().String())
	if se.s.GetEvent() != "" {
		buf.WriteString("|event=")
		buf.WriteString(se.s.GetEvent())
	}
	if se.s.GetMsg() != "" {
		buf.WriteString("|msg=")
		buf.WriteString(se.s.GetMsg())
	}
	return buf.String()
}

func (se *statusError) EvoStatus() *Status {
	return se.s
}

func Convert(err error) *Status {
	s, _ := FromError(err)
	return s
}

func FromError(err error) (*Status, bool) {
	if err == nil {
		return nil, true
	}
	type evostatus interface{ EvoStatus() *Status }
	var es evostatus
	if ok := errors.As(err, &es); ok {
		return es.EvoStatus(), true
	}
	return New(codes.Unknown, "").SetMsg(err.Error()), false
}
