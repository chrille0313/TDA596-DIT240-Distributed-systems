package mr

import "hash/fnv"

type Key = string
type Value = string
type KeyValue struct {
	Key   Key
	Value Value
}

func ihash(key Key) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}