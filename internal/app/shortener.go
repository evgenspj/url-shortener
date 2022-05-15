package app

import (
	"crypto/md5"
	"encoding/hex"
)

func GenShort(url string) string {
	md5Sum := md5.Sum([]byte(url))
	return hex.EncodeToString(md5Sum[:])
}
