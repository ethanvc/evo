package base

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

type Status struct {
	s internalStatus
}

func New(code codes.Code, event string) *Status {
	s := &Status{
		s: internalStatus{
			code:  code,
			event: event,
		},
	}
	return s
}

func (s *Status) SetMsg(format string, args ...any) *Status {
	s.s.msg = fmt.Sprintf(format, args...)
	return s
}

func (s *Status) GetCode() codes.Code {
	if s == nil {
		return codes.OK
	}
	return s.s.code
}

func (s *Status) GetMsg() string {
	if s == nil {
		return ""
	}
	return s.s.msg
}

func (s *Status) GetEvent() string {
	if s == nil {
		return ""
	}
	return s.s.event
}

func (s *Status) Err() StatusError {
	se := StatusError{
		s: s,
	}
	return se
}

func Code(err error) codes.Code {
	se, _ := err.(StatusError)
	return se.s.GetCode()
}

type internalStatus struct {
	code  codes.Code
	msg   string
	event string
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
