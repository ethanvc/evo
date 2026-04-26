package xobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"google.golang.org/grpc/codes"
)

var generateTraceIdFunc = GenerateTraceId
var generateSpanIdFunc = GenerateSpanId

var defaultSpan = newDefaultSpan()

var defaultHandler = NewJsonHandler(os.Stdout)

var defaultLogLevel = LevelInfo

var defaultGetLogLevel = GetLogLevel

func SetDefaultGetLogLevel(f GetAccessLogLevelFuncT) {
	defaultGetLogLevel = f
}

func GetDefaultGetLogLevel() GetAccessLogLevelFuncT {
	return defaultGetLogLevel
}

func SetDefaultLogLevel(lvl Level) {
	defaultLogLevel = lvl
}

func GetDefaultLogLevel() Level {
	return defaultLogLevel
}

// for test only
var sNow = time.Now

func SetDefaultSpan(span *Span) {
	defaultSpan = span
}

func SetGenerateTraceIdFunc(f func() string) {
	generateTraceIdFunc = f
	defaultSpan = newDefaultSpan()
}

func SetGenerateSpanIdFunc(f func(rootSpan bool) string) {
	generateSpanIdFunc = f
	defaultSpan = newDefaultSpan()
}

func GenerateTraceId() string {
	var buf [16]byte
	rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}

func GenerateSpanId(rootSpan bool) string {
	if rootSpan {
		return "0"
	}
	var buf [8]byte
	rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}

func newDefaultSpan() *Span {
	return NewSpan(context.Background(), &SpanConfig{
		Name: "default",
	})
}

func GetLogLevel(err error) Level {
	if err == nil {
		return LevelInfo
	}
	switch realErr := err.(type) {
	case *Error:
		switch realErr.GetCode() {
		case codes.OK, codes.NotFound, codes.AlreadyExists, codes.InvalidArgument, codes.Unauthenticated, codes.FailedPrecondition:
			return LevelInfo
		default:
			return LevelErr
		}
	default:
		return LevelErr
	}
}
