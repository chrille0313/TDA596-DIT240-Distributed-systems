package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	listenIp := flag.String("a", "0.0.0.0", "The IP address that the Chord client will bind to")
	publicIp := flag.String("aPublic", "", "The IP address that the Chord client will advertise to other nodes (defaults to the bind address)")
	port := flag.Int("p", 80, "Port to listen on")
	publicPort := flag.Int("pPublic", 0, "The port that will be advertised to other nodes (defaults to the listen port)")
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

	pubIp := *publicIp
	if pubIp == "" {
		pubIp = *listenIp
	}
	pubPort := *publicPort
	if pubPort == 0 {
		pubPort = *port
	}

	bindAddr := NodeAddress(net.JoinHostPort(*listenIp, fmt.Sprint(*port)))
	publicAddr := NodeAddress(net.JoinHostPort(pubIp, fmt.Sprint(pubPort)))

	node := MakeNode(bindAddr, publicAddr, *stabilizeInterval, *fixFingersInterval, *checkPredecessorInterval, *successorCount, id)
	
	if *joinAddress == "" && *joinPort == 0 {
		node.CreateRing()
	} else {
		node.JoinRing(NodeAddress(net.JoinHostPort(*joinAddress, fmt.Sprint(*joinPort))))
	}

	go node.Start()
	time.Sleep(500 * time.Millisecond) // Wait for RPC server to start

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
			if !filepath.IsAbs(path) {
				cwd, _ := os.Getwd()
				path = filepath.Join(cwd, path)
			}

			err := node.storeFile(path)
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

func (node *Node) lookup(filename string) error {
	key := hashString(filename)
	owner, err := node.findSuccessorIteratively(key, node.PublicAddress)
	if err != nil {
		return err
	}

	ownerID := owner.ID.String()
	ownerAddress := owner.Address

	var data []byte
	if ownerAddress == node.PublicAddress {
		var ok bool
		data, ok = node.StoredFiles[filename]
		if !ok {
			return fmt.Errorf("file %s not found locally", filename)
		}
	} else {
		reply, err := CallNodeRPC[RetrieveFileArgs, RetrieveFileReply](ownerAddress, "Node.RetrieveFile", &RetrieveFileArgs{Filename: filename})
		if err != nil {
			return err
		}
		if !reply.Found {
			return fmt.Errorf("file %s not found at owner", filename)
		}
		data = reply.Data
	}

	fmt.Printf("Owner ID: %s\nOwner Addr: %s\nContents:\n%s\n", ownerID, ownerAddress, string(data))
	return nil
}

func (node *Node) storeFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	filename := filepath.Base(filePath)
	key := hashString(filename)
	owner, err := node.findSuccessorIteratively(key, node.PublicAddress)
	if err != nil {
		return err
	}

	if owner.Address == node.PublicAddress {
		node.StoredFiles[filename] = data
		fmt.Printf("Stored %s locally (ID %s)\n", filename, node.ID.String())
		return nil
	}

	_, err = CallNodeRPC[StoreFileArgs, StoreFileReply](owner.Address, "Node.StoreFile", &StoreFileArgs{Filename: filename, Data: data})
	if err != nil {
		return err
	}

	fmt.Printf("Stored %s on %s (ID %s)\n", filename, owner.Address, owner.ID.String())
	return nil
}
