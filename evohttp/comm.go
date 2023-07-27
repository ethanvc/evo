package evohttp

import "fmt"

func assert(ok bool, f string, v ...any) {
	if ok {
		return
	}
	panic(fmt.Sprintf(f, v...))
}
