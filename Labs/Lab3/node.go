package main

import (
	"fmt"
	"math/big"
	"os"
	"time"
)

const (
	FingerTableSize = 161
	MaxHops         = 32
)

type Key string

type NodeAddress string

type IdPair struct {
	ID      *big.Int
	Address NodeAddress
}

type Node struct {
	ID               *big.Int
	BindAddress      NodeAddress
	PublicAddress    NodeAddress
	Predecessor      *IdPair
	Successors       []*IdPair
	FingerTable      []*IdPair

	StoredFiles map[string][]byte // Local file storage: filename -> file contents

	nextFingerToFix          int
	stabilizeInterval        time.Duration // --ts
	fixFingersInterval       time.Duration // --tff
	checkPredecessorInterval time.Duration // --tcp
}

func MakeNode(bindAddress NodeAddress, publicAddress NodeAddress, stabilizeInterval, fixFingersInterval, checkPredecessorInterval, successorCount int, identifier *big.Int) *Node {
	var id *big.Int

	if identifier != nil {
		id = identifier
	} else {
		id = computeNodeID(publicAddress)
	}

	return &Node{
		ID:                       id,
		BindAddress:              bindAddress,
		PublicAddress:            publicAddress,
		Successors:               make([]*IdPair, successorCount),
		FingerTable:              make([]*IdPair, FingerTableSize),
		StoredFiles:              make(map[string][]byte),
		nextFingerToFix:          1,
		stabilizeInterval:        time.Duration(stabilizeInterval) * time.Millisecond,
		fixFingersInterval:       time.Duration(fixFingersInterval) * time.Millisecond,
		checkPredecessorInterval: time.Duration(checkPredecessorInterval) * time.Millisecond,
	}
}

func computeNodeID(address NodeAddress) *big.Int {
	return hashString(string(address))
}

func (node *Node) Start() {
	node.startMaintenence()
	ListenRPC(string(node.BindAddress), node)
}

func (node *Node) CreateRing() error {
	node.Predecessor = nil
	for i := range node.Successors {
		node.Successors[i] = &IdPair{ID: node.ID, Address: node.PublicAddress}
	}
	return nil
}

func (node *Node) JoinRing(joinAdress NodeAddress) error {
	node.Predecessor = nil
	successor, err := node.findSuccessorIteratively(node.ID, joinAdress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "JoinRing: findSuccessor error: %v\n", err)
		return err
	} else if successor == nil {
		return fmt.Errorf("failed to find successor for join")
	}

	node.Successors[0] = successor

	reply, err := CallNodeRPC[GetSuccessorListArgs, GetSuccessorListReply](successor.Address, "Node.GetSuccessorList", &GetSuccessorListArgs{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "JoinRing: get successor list error: %v\n", err)
		return err
	}

	node.mergeSuccessorList(reply.Successors)
	
	return nil
}

func (node *Node) findSuccessorIteratively(id *big.Int, startAdress NodeAddress) (*IdPair, error) {
	found, nextNode := false, startAdress
	i := 0
	for !found && i < MaxHops {
		reply, err := CallNodeRPC[FindSuccessorArgs, FindSuccessorReply](nextNode, "Node.FindSuccessor", &FindSuccessorArgs{ID: id})
		if err != nil {
			fmt.Fprintf(os.Stderr, "findSuccessorIteratively: RPC error: %v\n", err)
			return nil, err
		}

		found, nextNode = reply.Found, reply.Successor
		i++
	}

	if found {
		return &IdPair{ID: computeNodeID(nextNode), Address: nextNode}, nil
	}

	return nil, fmt.Errorf("failed to find successor after %d hops", i)
}

func (node *Node) closestPrecedingNode(id *big.Int) (NodeAddress, error) {
	for i := len(node.FingerTable) - 1; i >= 1; i-- {
		fingerEntry := node.FingerTable[i]
		if fingerEntry != nil &&
			IsNodeAliveRPC(fingerEntry.Address) &&
			isBetween(node.ID, fingerEntry.ID, id, true) {
			return fingerEntry.Address, nil
		}
	}

	return node.popUntilAliveSuccessor().Address, nil
}

func (node *Node) stabilize() error {
	var lastErr error
	for attempts := 0; attempts < len(node.Successors); attempts++ {
		succ := node.popUntilAliveSuccessor()
		err := node.tryStabilizeWithSuccessor(succ)
		if err == nil {
			return nil
		}

		fmt.Fprintf(os.Stderr, "stabilize: error with successor %s: %v\n", succ.Address, err)
		lastErr = err
		node.Successors[0] = nil
		// On failure, loop to next alive successor
	}
	return lastErr
}

func (node *Node) tryStabilizeWithSuccessor(successor *IdPair) error {
	// Ask successor for its predecessor
	replyPred, err := CallNodeRPC[GetPredecessorArgs, GetPredecessorReply](successor.Address, "Node.GetPredecessor", &GetPredecessorArgs{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "stabilize: GetPredecessor error from %s: %v\n", successor.Address, err)
		return err
	}

	// Possibly adopt successor's predecessor if closer.
	if replyPred.Predecessor != nil && isBetween(node.ID, replyPred.Predecessor.ID, successor.ID, true) {
		node.Successors[0] = replyPred.Predecessor
		successor = node.Successors[0]
	}

	// Notify successor
	notifyArgs := &NotifyArgs{Node: &IdPair{ID: node.ID, Address: node.PublicAddress}}
	_, err = CallNodeRPC[NotifyArgs, NotifyReply](successor.Address, "Node.Notify", notifyArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stabilize: Notify error to %s: %v\n", successor.Address, err)
		return err
	}

	// Reconcile successor list from successor
	replySuccList, err := CallNodeRPC[GetSuccessorListArgs, GetSuccessorListReply](successor.Address, "Node.GetSuccessorList", &GetSuccessorListArgs{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "stabilize: GetSuccessorList error from %s: %v\n", successor.Address, err)
		return err
	}
	node.mergeSuccessorList(replySuccList.Successors)

	return nil
}

func (node *Node) popUntilAliveSuccessor() *IdPair {
	// Find first live successor; rotate list so it becomes head.
	liveIdx := -1
	for i, succ := range node.Successors {
		if succ != nil && IsNodeAliveRPC(succ.Address) {
			liveIdx = i
			break
		}
	}

	if liveIdx == -1 {
		// No live successors; reset list to self.
		self := &IdPair{ID: node.ID, Address: node.PublicAddress}
		for i := range node.Successors {
			node.Successors[i] = self
		}
		return self
	}

	if liveIdx > 0 {
		head := node.Successors[liveIdx]
		rest := append(node.Successors[liveIdx+1:], node.Successors[:liveIdx]...)
		node.Successors = append([]*IdPair{head}, rest...)
	}

	// Fill any nil slots (if present) with self as a safety net.
	for i := range node.Successors {
		if node.Successors[i] == nil {
			node.Successors[i] = &IdPair{ID: node.ID, Address: node.PublicAddress}
		}
	}

	return node.Successors[0]
}

func (node *Node) fixFingers() error {
	jumpAddress := jump(node.PublicAddress, node.nextFingerToFix)
	successor, err := node.findSuccessorIteratively(jumpAddress, node.PublicAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fixFingers: findSuccessor error: %v\n", err)
		return err
	}

	node.FingerTable[node.nextFingerToFix] = successor
	
	node.nextFingerToFix++
	if node.nextFingerToFix >= len(node.FingerTable) {
		node.nextFingerToFix = 1
	}

	return nil
}

func (node *Node) checkPredecessor() error {
	if node.Predecessor != nil && !IsNodeAliveRPC(node.Predecessor.Address) {
		node.Predecessor = nil
	}

	return nil
}

func (node *Node) mergeSuccessorList(successors []*IdPair) {
	for i := 1; i < len(node.Successors); i++ {
		if i-1 < len(successors) && successors[i-1] != nil {
			node.Successors[i] = successors[i-1]
		} else {
			// Fallback to self for empty slots.
			node.Successors[i] = &IdPair{ID: node.ID, Address: node.PublicAddress}
		}
	}
}

func (node *Node) startMaintenence() {
	go CallRepeatedly(node.stabilize, node.stabilizeInterval)
	go CallRepeatedly(node.fixFingers, node.fixFingersInterval)
	go CallRepeatedly(node.checkPredecessor, node.checkPredecessorInterval)
}

func (node *Node) PrintState() error {
	fmt.Println("====NODE====")
	fmt.Printf("Node ID: %s\n", node.ID.String())
	fmt.Printf("Node Bind Address: %s\n", node.BindAddress)
	fmt.Printf("Node Public Address: %s\n", node.PublicAddress)
	for filename := range node.StoredFiles {
		fmt.Printf("Stored file: %s\n", filename)
	}

	fmt.Println("====PREDECESSOR====")
	if node.Predecessor != nil {
		fmt.Printf("Predecessor ID: %s\n", node.Predecessor.ID.String())
		fmt.Printf("Predecessor Address: %s\n", node.Predecessor.Address)
	} else {
		fmt.Println("Predecessor: <nil>")
	}

	fmt.Println("====SUCCESSOR LIST====")
	for i, successor := range node.Successors {
		idStr := "<nil>"
		if successor != nil {
			idStr = successor.ID.String()
		}
		fmt.Printf("Successor %d ID: %s\n", i, idStr)

		addressStr := "<nil>"
		if successor != nil {
			addressStr = string(successor.Address)
		}
		fmt.Printf("Successor %d Address: %s\n", i, addressStr)
	}

	fmt.Println("====FINGER TABLE====")
	for i, finger := range node.FingerTable {
		idStr := "<nil>"
		if finger != nil {
			idStr = finger.ID.String()
		}

		addressStr := "<nil>"
		if finger != nil {
			addressStr = string(finger.Address)
		}

		fmt.Printf("Finger %d: ID: %s, Address: %s\n", i, idStr, addressStr)
	}

	return nil
}
