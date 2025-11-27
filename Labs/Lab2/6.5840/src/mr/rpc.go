//
// RPC definitions.
//
// remember to capitalize all names.
//

package mr

import (
	"os"
	"strconv"
)

type NoArgs struct{}

type TaskReply struct {
	Task       *Task
	MapData    *MapTaskData
	ReduceData *ReduceTaskData
}

type MapTaskData struct {
	File    string
	Buckets int
}

type ReduceTaskData struct {
	// TODO:
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
