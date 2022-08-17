package metric_gen

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
	letterIdxMax  = 63 / letterIdxBits
)

func fastRndString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)

	randomNumber, _ := rand.Int(rand.Reader, big.NewInt(63))
	for i, cache, remain := n-1, int(randomNumber.Int64()), letterIdxMax; i >= 0; {
		if remain == 0 {
			randomNumber, _ = rand.Int(rand.Reader, big.NewInt(63))
			cache, remain = int(randomNumber.Int64()), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}
