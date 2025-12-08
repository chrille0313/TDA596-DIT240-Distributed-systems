package chord

import (
	"crypto/sha1"
	"math/big"
	"time"
)

func CallRepeatedly(method func() error, timeBetweenCalls time.Duration) {
	method()
	time.Sleep(timeBetweenCalls)
}

func hashString(str string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(str))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}
