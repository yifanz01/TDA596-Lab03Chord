package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
)

//initial keyID and nodeID are m-length hash value
//then the m-length hash value mod 2^m ---> 0-63, ranged on the chord

// each node will hold a finger table with 6-length
// the i-th item of the finger table is nodeN+2^(i-1)
var fingerTableLen = 6

// the chord space is 2^6
// the identifier of nodes and keys are 6-length
var m = 6
var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(m)), nil) //2^6 = 64

type fingerItem struct {
	//todo: why bytes
	Identifier []byte //hash id, which is m-length, m=6 in this case
	//Identifier should be mapped into [0,(2^m-1)], on the chord with 2^m nodes in total
	//todo:IP address?
	Addr string //address of node
}

type Node struct {
	Name       string   // can be address or user-defined name which is given by the input args
	Addr       string   // IP:Port
	Identifier *big.Int //chord space identifier,0-63

	FingerTable []fingerItem
	nextFinger  int //the index of the next finger, [0,m-1]

	PredecessorAddr string
	//the size of successor list is given by the input argument
	SuccessorsAddr []string

	mutex sync.Mutex

	// Create bucket in form of map
	Bucket map[*big.Int]string
	Backup map[*big.Int]string
}

type StoreFileRPCReply struct {
	Success bool
	Err     error
	Backup  bool
}

// the first node in the chord, no predecessor, all the successors are the node itself
func (node *Node) createNewChord() {
	node.PredecessorAddr = ""
	for i := 0; i < len(node.SuccessorsAddr); i++ {
		node.SuccessorsAddr[i] = node.Addr
	}
}

// NewNode create a new node, and assign the initial values to it's attributes
func NewNode(args Arguments) *Node {
	//assign address to the new node
	newNode := &Node{}
	var nodeAddr string
	if args.IpAddress == "127.0.0.1" || args.IpAddress == "localhost" {
		nodeAddr = "127.0.0.1"
	} else if args.IpAddress == "0.0.0.0" {
		nodeAddr = getIP()
	} else {
		nodeAddr = getLocalAddress()
	}
	newNode.Addr = fmt.Sprintf("%s:%d", nodeAddr, args.Port)

	//assign name to the new node
	if args.ClientName == "Default" {
		newNode.Name = newNode.Addr
	} else {
		newNode.Name = args.ClientName
	}

	//0-63
	newNode.Identifier = StrHash(newNode.Name)
	newNode.Identifier.Mod(newNode.Identifier, hashMod)

	newNode.FingerTable = make([]fingerItem, fingerTableLen+1)

	//todo:store file and backup

	newNode.PredecessorAddr = ""
	newNode.SuccessorsAddr = make([]string, args.R)

	//initiate id to n+2^(i-1), all addr to node.Addr
	newNode.initFingerTable()
	//initiate all to empty string
	newNode.initSuccessorsAddr()

	rootPath := "../files/" + newNode.Name
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		err := os.MkdirAll(rootPath, os.ModePerm)
		if err != nil {
			log.Println("failed to create file: " + rootPath + err.Error())
		} else {

			fileMode := []string{"/upload", "/download", "/chord"}
			for _, mode := range fileMode {
				//create upload/download/chord folder for a certain node
				if _, err := os.Stat(rootPath + mode); os.IsNotExist(err) {
					err := os.Mkdir(rootPath+mode, os.ModePerm)
					if err != nil {
						log.Println("failed to create file of: " + rootPath + mode + err.Error())
					}
				} else {
					log.Println("the file of :" + rootPath + mode + " already exist")
				}
			}

		}
		//todo:generateRSAKey

	} else {
		fmt.Println("the node folder of" + rootPath + " already exist")
		//todo:file operations
	}

	return newNode
}

// m rows
// each row contains key-id, successor(key-id),interval[]
func (node *Node) initFingerTable() {
	node.FingerTable[0].Identifier = node.Identifier.Bytes()
	node.FingerTable[0].Addr = node.Addr
	fmt.Println("fingerTable[0] of node-", node.Name, " is:", node.FingerTable[0].Identifier, node.FingerTable[0].Addr)
	//add rows in finger table
	for i := 1; i < fingerTableLen+1; i++ {
		identifier := new(big.Int).Add(node.Identifier, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(i)-1), nil))
		identifier.Mod(identifier, hashMod)
		node.FingerTable[i].Identifier = identifier.Bytes()
		//todo: what is the address here exactly
		node.FingerTable[i].Addr = node.Addr
	}
}

func (node *Node) initSuccessorsAddr() {
	successorsAddrNum := len(node.SuccessorsAddr)
	for i := 0; i < successorsAddrNum; i++ {
		node.SuccessorsAddr[i] = ""
	}
}

func (node *Node) joinChord(joinNodeAddr string) error {
	log.Printf("Node %s wanna join the Chord: %s", node.Addr, joinNodeAddr)
	node.PredecessorAddr = ""

	//find the successor of node and store it in index-0
	var reply FindSuccessorRPCReply
	err := ChordCall(joinNodeAddr, "Node.FindSuccessorRPC", node.Identifier, &reply)
	if err != nil {
		return err
	}
	node.SuccessorsAddr[0] = reply.SuccessorAddress

	//node is the predecessor of node.Successor
	err = ChordCall(node.SuccessorsAddr[0], "Node.NotifyRPC", node.Addr, &reply)
	if err != nil {
		return err
	}

	return nil
}
