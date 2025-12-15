package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
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

	var id *big.Int = nil
	if *identifier != "" {
		var ok bool
		id, ok = new(big.Int).SetString(*identifier, 10)
		if !ok {
			log.Fatalf("Invalid identifier: %s", *identifier)
		}
	}

	node := MakeNode(NodeAddress(net.JoinHostPort(*listenIp, fmt.Sprint(*port))), *stabilizeInterval, *fixFingersInterval, *checkPredecessorInterval, *successorCount, id)
	
	if *joinAddress == "" && *joinPort == 0 {
		node.CreateRing()
	} else {
		node.JoinRing(NodeAddress(net.JoinHostPort(*joinAddress, fmt.Sprint(*joinPort))))
	}

	go node.Start()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		scanner.Scan()
		err := scanner.Err()
		if err != nil {
			log.Fatal(err)
		}

		input := scanner.Text()
		args := strings.Split(input, " ")

		if len(args) == 0 {
			fmt.Println("Invalid input!")
			continue
		}

		command := args[0]

		switch command {
		case "Lookup":
			if len(args) < 2 {
				fmt.Println("No file provided!")
				continue
			}

			file := args[1]
			err := node.lookup(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "StoreFile":
			if len(args) < 2 {
				fmt.Println("No file path provided!")
				continue
			}

			path := args[1]
			err := storeFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "PrintState":
			err := node.PrintState()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "Quit":
			os.Exit(0)

		case "q":
			os.Exit(0)

		default:
			fmt.Println("Unknown command!")
		}
	}
}

// ‘Lookup’ takes as input the name of a file to be searched (e.g., “Hello.txt”).
//
//	The Chord client takes this string, hashes it to a key in the identifier space,
//	and performs a search for the node that is the successor to the key (i.e., the owner of the key).
//	The Chord client then outputs that node’s identifier, IP address, port, and the contents of the file.
func (node *Node) lookup(filename string) error {
	key := hashString(filename)
	node.findSuccessorIteratively(key, node.Address)
	return nil
}

// 'StoreFile' takes the location of a file on a local disk, then performs a lookup to find the Chord
//
//	node to store the file at, then uploading the file to the Chord ring.
func storeFile(filePath string) error {
	return nil
}
