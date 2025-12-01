package mr

import (
	"log"
	"os"
)

var debugEnabled = os.Getenv("MR_DEBUG") != ""

func debugf(format string, args ...interface{}) {
	if !debugEnabled {
		return
	}
	log.Printf(format, args...)
}