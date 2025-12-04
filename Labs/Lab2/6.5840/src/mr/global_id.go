package mr

var globalID int = 0

func getGlobalID() int {
	globalID++
	return globalID
}
