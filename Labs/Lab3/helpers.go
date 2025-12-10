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

func isBetween(from, id, to *big.Int) bool {
    if from.Cmp(to) < 0 {
        return from.Cmp(id) < 0 && id.Cmp(to) < 0
    }
    // Wrap-around case
    return id.Cmp(from) > 0 || id.Cmp(to) < 0
}