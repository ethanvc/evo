package base

import "runtime"

func GetCaller(skip int) uintptr {
	var pcs [1]uintptr
	runtime.Callers(skip+2, pcs[:])
	return pcs[0]
}

func GetCallerFrame(pc uintptr) runtime.Frame {
	fs := runtime.CallersFrames([]uintptr{pc})
	frame, _ := fs.Next()
	return frame
}
