package main

import "math/big"

type FindSuccessorArgs struct {
	ID *big.Int
}

type FindSuccessorReply struct {
	Found     bool
	Successor NodeAddress
}

func (node *Node) FindSuccessor(args *FindSuccessorArgs, reply *FindSuccessorReply) error {
	nodeSuccessor := node.popUntilAliveSuccessor()
	if isBetween(node.ID, args.ID, nodeSuccessor.ID, true) {
		reply.Found = true
		reply.Successor = nodeSuccessor.Address
		return nil
	}

	reply.Found = false
	successor, err := node.closestPrecedingNode(args.ID)
	if err != nil {
		return err
	}
	reply.Successor = successor

	return nil
}

type GetPredecessorArgs struct{}

type GetPredecessorReply struct {
	Predecessor *IdPair
}

func (node *Node) GetPredecessor(args *GetPredecessorArgs, reply *GetPredecessorReply) error {
	if node.Predecessor != nil {
		reply.Predecessor = node.Predecessor
	}
	return nil
}

type GetSuccessorListArgs struct{}

type GetSuccessorListReply struct {
	Successors []*IdPair
}

func (node *Node) GetSuccessorList(args *GetSuccessorListArgs, reply *GetSuccessorListReply) error {
	node.popUntilAliveSuccessor()
	reply.Successors = make([]*IdPair, 0, len(node.Successors))
	for i := range node.Successors {
		reply.Successors = append(reply.Successors, node.Successors[i])
	}
	return nil
}

type IsAliveArgs struct{}

type IsAliveReply struct{}

func (node *Node) IsAlive(args *IsAliveArgs, reply *IsAliveReply) error {
	return nil
}

type NotifyArgs struct {
	Node *IdPair
}

type NotifyReply struct{}

func (node *Node) Notify(args *NotifyArgs, reply *NotifyReply) error {
	if node.Predecessor == nil || isBetween(node.Predecessor.ID, args.Node.ID, node.ID, true) {
		node.Predecessor = args.Node
	}
	return nil
}

type StoreFileArgs struct {
	Filename string
	Data     []byte
}

type StoreFileReply struct {}

func (node *Node) StoreFile(args *StoreFileArgs, reply *StoreFileReply) error {
	node.StoredFiles[args.Filename] = args.Data
	return nil
}

type RetrieveFileArgs struct {
	Filename string
}

type RetrieveFileReply struct {
	Found bool
	Data  []byte
}

func (node *Node) RetrieveFile(args *RetrieveFileArgs, reply *RetrieveFileReply) error {
	data, ok := node.StoredFiles[args.Filename]
	if !ok {
		reply.Found = false
		return nil
	}
	reply.Found = true
	reply.Data = data
	return nil
}

