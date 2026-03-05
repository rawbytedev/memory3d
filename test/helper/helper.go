package helper

import "crypto/rand"

func GenerateData(size uint32) []byte {
	tmp := make([]byte, size)
	rand.Read(tmp)
	return tmp
}
