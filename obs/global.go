package obs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"
)

var generateTraceIdFunc = GenerateTraceId
var generateSpanIdFunc = GenerateSpanId

var defaultSpan = newDefaultSpan()

var defaultReporter = newReporter()

var defaultHandler LogHandler = NewJsonHandler(os.Stdout)

func SetDefaultHandler(handler LogHandler) {
	if handler == nil {
		panic("handler is nil")
	}
	defaultHandler.Flush()
	_ = defaultHandler.Close()
	defaultHandler = handler
}

var defaultLogLevel = LevelInfo

var defaultGetLogLevelAndEvent = GetLogLevelAndEvent

func SetDefaultGetLogLevel(f GetLogLvlAndEventFuncT) {
	defaultGetLogLevelAndEvent = f
}

func GetDefaultGetLogLevelAndEvent() GetLogLvlAndEventFuncT {
	return defaultGetLogLevelAndEvent
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
		Method: "default",
	})
}

func GetLogLevelAndEvent(err error) (Level, string) {
	if err == nil {
		return LevelInfo, OK.String()
	}
	switch realErr := err.(type) {
	case *Error:
		switch realErr.GetCode() {
		case OK, NotFound, AlreadyExists, InvalidArgument, Unauthenticated, FailedPrecondition:
			return LevelInfo, realErr.GetCode().String()
		default:
			return LevelErr, realErr.GetCode().String()
		}
	default:
		return LevelErr, Internal.String()
	}
}
