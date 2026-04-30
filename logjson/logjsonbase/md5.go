package logjsonbase

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
)

func LogMd5(msg []byte) string {
	// "len=" (4) + max int64 digits (20) + "," (1) + md5 hex (32) = 57
	var buf [57]byte
	b := append(buf[:0], "len="...)
	b = strconv.AppendInt(b, int64(len(msg)), 10)
	b = append(b, ',')
	h := md5.Sum(msg)
	b = hex.AppendEncode(b, h[:])
	return string(b)
}
