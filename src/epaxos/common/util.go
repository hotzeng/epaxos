package common

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	math_rand "math/rand"
	"os"
)

func InitializeRand() {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		panic("cannot seed math/rand package with cryptographically secure random number generator")
	}
	seed := binary.LittleEndian.Uint64(b[:])
	math_rand.Seed(int64(seed))
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
