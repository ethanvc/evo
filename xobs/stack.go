package xobs

import (
	"fmt"
	"runtime"
	"strings"
)

func GetPanicPosition(skip int) string {
	var pcs [10]uintptr
	const ParentSkipCount = 4
	cnt := runtime.Callers(skip+ParentSkipCount, pcs[:])
	if cnt == 0 {
		return "RuntimePanic;CallersReturnNothing"
	}
	realPcs := pcs[0:cnt]
	frames := runtime.CallersFrames(realPcs)

	more := true
	var frame runtime.Frame
	for {
		if !more {
			return "RuntimePanic;NotFoundBusinessCode"
		}
		frame, more = frames.Next()
		if strings.Contains(frame.File, "/src/runtime/") {
			continue
		}
		break
	}
	const keepTailPart = 2
	s := GetFilePathTailPart(frame.File, keepTailPart)
	return fmt.Sprintf("RuntimePanic;%s:%d;", s, frame.Line)
}

func GetFilePathTailPart(filePath string, count int) string {
	currentCnt := 0
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' || filePath[i] == '\\' {
			currentCnt++
			if currentCnt >= count {
				return filePath[i+1:]
			}
		}
	}
	return filePath
}
