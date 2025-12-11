package main

import (
	"crypto/sha1"
	"math/big"
	"time"
)

const keySize = sha1.Size * 8

var two = big.NewInt(2)
var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(keySize), nil)

func CallRepeatedly(method func() error, timeBetweenCalls time.Duration) {
	method()
	time.Sleep(timeBetweenCalls)
}

func hashString(str string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(str))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

func isBetween(start, id, end *big.Int, inclusive bool) bool {
	if end.Cmp(start) > 0 {
		return (start.Cmp(id) < 0 && id.Cmp(end) < 0) || (inclusive && id.Cmp(end) == 0)
	} else {
		return start.Cmp(id) < 0 || id.Cmp(end) < 0 || (inclusive && id.Cmp(end) == 0)
	}
}

func jump(address NodeAddress, fingerentry int) *big.Int {
	n := hashString(string(address))
	fingerentryminus1 := big.NewInt(int64(fingerentry) - 1)
	jump := new(big.Int).Exp(two, fingerentryminus1, nil)
	sum := new(big.Int).Add(n, jump)

	return new(big.Int).Mod(sum, hashMod)
}
