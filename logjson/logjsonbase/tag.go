package logjsonbase

import "strings"

type TagOptions struct {
	MD5     bool
	Discard bool
}

func ParseTag(tag string) TagOptions {
	var opts TagOptions
	for _, opt := range strings.Split(tag, ",") {
		switch strings.TrimSpace(opt) {
		case "md5":
			opts.MD5 = true
		case "discard":
			opts.Discard = true
		}
	}
	return opts
}
