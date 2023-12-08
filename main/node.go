package main

import (
	"fmt"
	"math/big"
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
	Name       string   // e.g. N-5
	Addr       string   // IP:Port
	Identifier *big.Int //chord space identifier,0-63

	FingerTable []fingerItem
	nextFinger  int //the index of the next finger, [0,m-1]

	PredecessorAddr string
	SuccessorsAddr  []string

	mutex sync.Mutex
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

func (node *Node) createNewChord() {
	node.PredecessorAddr = ""
	for i := 0; i < len(node.SuccessorsAddr); i++ {
		//todo: why set successor to address
		node.SuccessorsAddr[i] = node.Addr
	}
}
