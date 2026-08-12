package requestid

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func New() string {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return time.Now().UTC().Format("20060102T150405.000000000")
	}
	return time.Now().UTC().Format("20060102T150405.000") + "-" + hex.EncodeToString(random[:])
}
