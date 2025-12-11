package main

import (
	"flag"
	"fmt"
	"net"
)

// The Chord client will open a TCP socket and listen for incoming connections on port specified by -p.
// If neither --ja nor --jp is specified, then the Chord client starts a new ring by invoking ‘create’.
// The Chord client will initialize the successor list and finger table appropriately (i.e., all will point
// to the client itself).
// Otherwise, the Chord client joins an existing ring by connecting to the Chord client specified by --ja
//  and --jp and invoking ‘join’. The initial steps the Chord client takes when joining the network are
// described in detail in Section IV.E.1 “Node Joins and Stabilization” of the Chord paper.
// Periodically, the Chord client will invoke various stabilization routines in order to handle nodes
// joining and leaving the network. The Chord client will invoke ‘stabilize’, ‘fix fingers’, and
// ‘check predecessor’ every --ts, --tff, and --tcp milliseconds, respectively.

func main() {
	listenIp := flag.String("a", "0.0.0.0", "The IP address that the Chord client will bind to, as well as advertise to other nodes")
	port := flag.Int("p", 80, "Port to listen on")
	joinAddress := flag.String("ja", "", "The IP address of the machine running a Chord node. The Chord client will join this node's ring.")
	joinPort := flag.Int("jp", 0, "The port that an existing Chord node is bound to and listening on. The Chord client will join this node's ring.")
	stabilizeInterval := flag.Int("ts", 1000, "The time in milliseconds between invocations of 'stabilize'")
	fixFingersInterval := flag.Int("tff", 1000, "The time in milliseconds between invocations of 'fix fingers'")
	checkPredecessorInterval := flag.Int("tcp", 1000, "The time in milliseconds between invocations of 'check predecessor'")
	successorCount := flag.Int("r", 3, "The number of successors maintained by the Chord client")
	identifier := flag.String("i", "", "The identifier (ID) assigned to the Chord client which will override the ID computed by the SHA1 sum of the client's IP address and port number")
	flag.Parse()

	node := MakeNode(NodeAddress(net.JoinHostPort(*listenIp, fmt.Sprint(*port))), *stabilizeInterval, *fixFingersInterval, *checkPredecessorInterval, *successorCount, identifier)

	if *joinAddress != "" && *joinPort != 0 {
		node.CreateRing()
	} else {
		node.JoinRing(NodeAddress(net.JoinHostPort(*joinAddress, fmt.Sprint(*joinPort))))
	}

	node.Start()
}

// ‘Lookup’ takes as input the name of a file to be searched (e.g., “Hello.txt”).
//
//	The Chord client takes this string, hashes it to a key in the identifier space,
//	and performs a search for the node that is the successor to the key (i.e., the owner of the key).
//	The Chord client then outputs that node’s identifier, IP address, port, and the contents of the file.
func lookup() error {
	return nil
}

// 'StoreFile' takes the location of a file on a local disk, then performs a lookup to find the Chord
//
//	node to store the file at, then uploading the file to the Chord ring.
func storeFile() error {
	return nil
}

// ‘PrintState’ requires no input. The Chord client outputs its local state information at the current time, which consists of:
// The Chord client’s own node information and its stored files,
// The node information for all nodes in the successor list,
// The node information for all nodes in the finger table,
// where “node information” corresponds to the identifier, IP address, and port for a given node.
func printState() error {
	return nil
}
