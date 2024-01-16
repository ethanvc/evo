package rediskey

import "fmt"

func UserCacheKey(uid int64) string {
	return fmt.Sprintf("a/%d", uid)
}
