package base

import (
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
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

func (s *Status) Err() StatusError {
	se := StatusError{
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
	se, ok := err.(StatusError)
	if !ok {
		return codes.Unknown
	}
	return se.s.GetCode()
}

func NotFound(err error) bool {
	return Code(err) == codes.NotFound
}

type internalStatus struct {
	Code  codes.Code `json:"code"`
	Msg   string     `json:"msg,omitempty"`
	Event string     `json:"event,omitempty"`
}

type StatusError struct {
	s *Status
}

func (se StatusError) Status() *Status {
	return se.s
}

func (se StatusError) Error() string {
	return ""
}

func StatusFromError(err error) *Status {
	if err == nil {
		return nil
	}
	if s, ok := err.(StatusError); ok {
		return s.s
	}
	return New(codes.Internal, "NonStandardError")
}
