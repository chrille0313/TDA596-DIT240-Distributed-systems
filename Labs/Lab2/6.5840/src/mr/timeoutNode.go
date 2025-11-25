package mr

import "time"

type TimeoutNode struct {
	taskID   ID
	deadline time.Time
	next     *TimeoutNode
}

type TimeoutList struct {
	head *TimeoutNode
}

func (l *TimeoutList) Insert(taskID ID, deadline time.Time) {
	node := &TimeoutNode{taskID: taskID, deadline: deadline}
	if l.head == nil || deadline.Before(l.head.deadline) {
		node.next = l.head
		l.head = node
		return
	}
	prev := l.head
	for prev.next != nil && !deadline.Before(prev.next.deadline) {
		prev = prev.next
	}
	node.next = prev.next
	prev.next = node
}

func (l *TimeoutList) PopExpired(now time.Time) []ID {
	var expired []ID

	for l.head != nil && !now.Before(l.head.deadline) {
		expired = append(expired, l.head.taskID)
		l.head = l.head.next
	}

	return expired
}
