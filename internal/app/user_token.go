package app

import (
	"encoding/binary"
	"encoding/hex"
)

func GetUserIDFromToken(token string) uint32 {
	data, err := hex.DecodeString(token)
	if err != nil {
		panic(err)
	}
	userID := binary.BigEndian.Uint32(data[:4])
	return userID
}
