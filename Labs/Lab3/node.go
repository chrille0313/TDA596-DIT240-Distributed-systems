package main

import (
	"fmt"
	"math/big"
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
	ID          *big.Int
	Address     NodeAddress
	FingerTable []IdPair
	Predecessor *IdPair
	Successor   IdPair            // TODO: Change to list of successors
	StoredFiles map[string][]byte // Local file storage: filename -> file contents

	nextFingerToFix          int
	stabilizeInterval        time.Duration // --ts
	fixFingersInterval       time.Duration // --tff
	checkPredecessorInterval time.Duration // --tcp
}

func MakeNode(address NodeAddress, stabilizeInterval, fixFingersInterval, checkPredecessorInterval, successorCount int, identifier string) *Node {
	if identifier != "" {
		address = NodeAddress(identifier)
	}

	return &Node{
		ID:                       computeNodeID(address),
		Address:                  address,
		FingerTable:              make([]IdPair, FingerTableSize),
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
	// TODO: Implement
	go node.startMaintenence()
	StartRPCServer(string(node.Address), node)
}

func (node *Node) CreateRing() error {
	node.Predecessor = nil
	node.Successor = IdPair{ID: node.ID, Address: node.Address}
	return nil
}

func (node *Node) JoinRing(joinAdress NodeAddress) error {
	node.Predecessor = nil
	successor, err := node.findSuccessorIteratively(node.ID, joinAdress)
	fmt.Printf("join successor: %v\n", successor)
	if err != nil {
		fmt.Printf("errore: %v \n", err)
		return err
	}
	node.Successor = IdPair{ID: computeNodeID(successor), Address: successor}
	return nil
}

type FindSuccessorArgs struct {
	ID *big.Int
}

type FindSuccessorReply struct {
	found     bool
	Successor NodeAddress
}

func (node *Node) FindSuccessor(args *FindSuccessorArgs, reply *FindSuccessorReply) error {
	if isBetween(node.ID, args.ID, node.Successor.ID, true) {
		reply.found = true
		reply.Successor = node.Successor.Address
		return nil
	}

	reply.found = false
	successor, err := node.closestPrecedingNode(args.ID)
	if err != nil {
		return err
	}
	reply.Successor = successor

	return nil
}

func (node *Node) findSuccessorIteratively(id *big.Int, startAdress NodeAddress) (NodeAddress, error) {
	found, nextNode := false, startAdress
	i := 0
	for !found && i < MaxHops {
		args := &FindSuccessorArgs{ID: id}
		reply, err := CallNodeRPC[FindSuccessorArgs, FindSuccessorReply](nextNode, "Node.FindSuccessor", args)
		if err != nil {
			return "", err
		}

		found, nextNode = reply.found, reply.Successor
		i++
	}

	if found {
		return nextNode, nil
	}

	return "", nil // TODO: "report error"
}

func (node *Node) closestPrecedingNode(id *big.Int) (NodeAddress, error) {
	for i := len(node.FingerTable) - 1; i > 0; i-- {
		fingerEntry := node.FingerTable[i]
		if isBetween(node.ID, fingerEntry.ID, id, false) {
			return fingerEntry.Address, nil
		}
	}

	return node.Successor.Address, nil
}

type GetPredecessorArgs struct{}

type GetPredecessorReply struct {
	Predecessor *NodeAddress
}

func (node *Node) GetPredecessor(args *GetPredecessorArgs, reply *GetPredecessorReply) error {
	if node.Predecessor != nil {
		reply.Predecessor = &node.Predecessor.Address
	}
	return nil
}

func (node *Node) stabilize() error {
	args := &GetPredecessorArgs{}
	reply, err := CallNodeRPC[GetPredecessorArgs, GetPredecessorReply](node.Successor.Address, "Node.GetPredecessor", args)
	if err != nil {
		return err
	}
	x := reply.Predecessor

	if x != nil {
		xID := computeNodeID(*x)
		if isBetween(node.ID, xID, node.Successor.ID, false) {
			node.Successor = IdPair{ID: xID, Address: *x}
		}
	}

	notifyArgs := &NotifyArgs{ID: node.ID, Address: node.Address}
	_, err = CallNodeRPC[NotifyArgs, NotifyReply](node.Successor.Address, "Node.Notify", notifyArgs)
	return err
}

type NotifyArgs struct {
	ID      *big.Int
	Address NodeAddress
}

type NotifyReply struct{}

func (node *Node) Notify(args *NotifyArgs, reply *NotifyReply) error {
	if node.Predecessor == nil || isBetween(node.Predecessor.ID, args.ID, node.ID, false) {
		node.Predecessor = &IdPair{ID: args.ID, Address: args.Address}
	}
	return nil
}

func (node *Node) fixFingers() error {
	jumpAddress := jump(node.Address, node.nextFingerToFix)
	successor, err := node.findSuccessorIteratively(jumpAddress, node.Address)
	if err != nil {
		// fmt.Printf("errore: %v \n", err)
		return err
	}
	// fmt.Printf("successor; %v", successor)
	node.FingerTable[node.nextFingerToFix] = IdPair{ID: computeNodeID(successor), Address: successor}

	node.nextFingerToFix++
	if node.nextFingerToFix >= FingerTableSize {
		node.nextFingerToFix = 1
	}

	return nil
}

func (node *Node) checkPredecessor() error {
	if node.Predecessor == nil {
		return nil
	}

	args := &GetPredecessorArgs{}

	// Attempt RPC to predecessor
	_, err := CallNodeRPC[GetPredecessorArgs, GetPredecessorReply](node.Predecessor.Address, "Node.GetPredecessor", args)

	// If RPC fails, predecessor has crashed
	if err != nil {
		node.Predecessor = nil
	}

	return nil
}

func (node *Node) startMaintenence() {
	for {
		// CallRepeatedly(node.stabilize, node.stabilizeInterval)
		node.stabilize()
		time.Sleep(node.stabilizeInterval)
		// CallRepeatedly(node.fixFingers, node.fixFingersInterval)
		// node.fixFingers()
		// time.Sleep(node.fixFingersInterval)
		// CallRepeatedly(node.checkPredecessor, node.checkPredecessorInterval)
		node.checkPredecessor()
		time.Sleep(node.checkPredecessorInterval)
	}
}

// ‘PrintState’ requires no input. The Chord client outputs its local state information at the current time, which consists of:
// The Chord client’s own node information and its stored files,
// The node information for all nodes in the successor list,
// The node information for all nodes in the finger table,
// where “node information” corresponds to the identifier, IP address, and port for a given node.
func (node *Node) PrintState() error {

	// The Chord client’s own node information and its stored files
	fmt.Println("====NODE====")
	fmt.Printf("Node ID: %s\n", node.ID.String())
	fmt.Printf("%s\n", node.Address)
	for filename := range node.StoredFiles {
		fmt.Printf("Stored file: %s\n", filename)
	}

	fmt.Println("====PREDECESSOR====")
	if node.Predecessor != nil {
		fmt.Printf("Predecessor ID: %s\n", node.Predecessor.ID.String())
		fmt.Printf("%s\n", node.Predecessor.Address)
	} else {
		fmt.Println("Predecessor: <nil>")
	}

	// The node information for all nodes in the successor list,
	fmt.Println("====SUCCESSOR LIST====")
	fmt.Printf("Successor ID: %s\n", node.Successor.ID.String())
	fmt.Printf("%s\n", node.Successor.Address)

	// The node information for all nodes in the finger table
	fmt.Println("====FINGER TABLE====")
	for i, finger := range node.FingerTable {
		fmt.Printf("Finger %d: ID: %s, Address: %s\n", i, finger.ID.String(), finger.Address)
	}

	return nil
}
