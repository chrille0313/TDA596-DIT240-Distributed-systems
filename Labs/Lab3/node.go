package chord

import (
	"math/big"
	"time"
)

type Key string

type NodeAddress string

type Node struct {
	ID          *big.Int
	Address     NodeAddress
	FingerTable []NodeAddress
	Predecessor *NodeAddress
	Successors  map[*big.Int]NodeAddress

	Bucket map[Key]string

	stabilizeInterval        time.Duration // --ts
	fixFingersInterval       time.Duration // --tff
	checkPredecessorInterval time.Duration // --tcp
}

func MakeNode(address NodeAddress, stabilizeInterval, fixFingersInterval, checkPredecessorInterval, successorCount int, identifier string) *Node {
	return &Node{
		ID:                       computeNodeID(address, identifier),
		Address:                  address,
		Successors:               make(map[*big.Int]NodeAddress, successorCount),
		stabilizeInterval:        time.Duration(stabilizeInterval) * time.Millisecond,
		fixFingersInterval:       time.Duration(fixFingersInterval) * time.Millisecond,
		checkPredecessorInterval: time.Duration(checkPredecessorInterval) * time.Millisecond,
	}
}

func computeNodeID(address NodeAddress, identifier string) *big.Int {
	if identifier != "" {
		return hashString(identifier)
	} else {
		return hashString(string(address))
	}
}

func (node *Node) Start() (NodeAddress, error) {
	// TODO: Implement
	node.startMaintenence()

	return "", nil
}

func (node *Node) CreateRing() error {
	node.Predecessor = nil
	node.Successors[0] = node.Address
	return nil
}

func (node *Node) JoinRing(joinAdress NodeAddress) error {
	node.Predecessor = nil
	successor, err := node.findSuccessorRPC(joinAdress, node.ID)
	if err != nil {
		return err
	}
}

type FindSuccessorArgs struct {
	ID *big.Int
}

type FindSuccessorReply struct {
	found     bool
	Successor NodeAddress
}

func (node *Node) Successor(args *FindSuccessorArgs, reply *FindSuccessorReply) error {
	successor, exists := node.Successors[args.ID]

	if exists {
		reply.found = true
		reply.Successor = successor
	} else {
		reply.found = false
		successor, err := node.closestPrecedingNode(args.ID)
		if err != nil {
			return err
		}
		reply.Successor = successor
	}

	return nil
}

func (node *Node) findSuccessor(id *big.Int, address NodeAddress) (NodeAddress, error) {
	found, nextNode := false, address
	i := 0
	for !found { // TODO: max hops
		args := &FindSuccessorArgs{ID: id}
		reply := &FindSuccessorReply{}
		err := CallRPC(string(nextNode), "Node.Successor", args, reply)
		if err != nil {
			return "", err
		}

		found, nextNode = reply.found, reply.Successor
		i++
	}

	if found {
		return nextNode, nil
	}

	return "", nil
}



func (node *Node) closestPrecedingNode(id *big.Int) (NodeAddress, error) {
	// TODO: Implement
	// skip this loop if you do not have finger tables implemented yet
        // for i = m downto 1
        //     if (finger[i] ∈ (n,id))
        //         return finger[i];
        // return successor;
	return "", nil
}

type GetPredecessorArgs struct{}

type GetPredecessorReply struct {
    Predecessor NodeAddress
}

func (node *Node) GetPredecessor(args *GetPredecessorArgs, reply *GetPredecessorReply) error {
 	if node.Predecessor != nil {
        reply.Predecessor = *node.Predecessor
    }
    return nil
}

func (node *Node) stabilize() error {
	var successor NodeAddress
    for _, addr := range node.Successors {
        successor = addr
        break
    }	

	if successor == "" {
        return nil
    }

	args := &GetPredecessorArgs{}
    reply := &GetPredecessorReply{}
    err := CallRPC(string(successor), "Node.GetPredecessor", args, reply)
    if err != nil {
        return err
    }

	x := reply.Predecessor // the sucessors predecessor


	// if x != "" && isBetween(node.ID, hashString(string(x)), hashString(string(successor))) {
    //     // Update successor to x
	// 	node.Successors[big.NewInt(0)] = x
    //     successor = x
    // }



	notifyArgs := &NotifyArgs{ID: node.ID, Address: node.Address}
    notifyReply := &NotifyReply{}
    return CallRPC(string(successor), "Node.Notify", notifyArgs, notifyReply)
}

type NotifyArgs struct {
    ID      *big.Int
    Address NodeAddress
}

type NotifyReply struct{}

func (node *Node) checkPredecessor() error {
	return nil
}

func (node *Node) fixFingers() error {
	return nil
}

func (node *Node) startMaintenence() {
	go CallRepeatedly(node.stabilize, node.stabilizeInterval)
	go CallRepeatedly(node.fixFingers, node.fixFingersInterval)
	go CallRepeatedly(node.checkPredecessor, node.checkPredecessorInterval)
}
