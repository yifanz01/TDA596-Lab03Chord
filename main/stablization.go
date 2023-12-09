package main

import (
	"log"
	"math/big"
)

func (node *Node) stablize() error {
	//firstly, update successor list
	//the successor list of node: successor[0] is the next server node that near active node
	//1-(n-1) are the first (n-1) items of the successor list of successor[0]
	//if successor[0] is dead, remove it and shift the successor list to the upper
	var getSuccessorListRPCReply GetSuccessorListRPCReply
	err := ChordCall(node.SuccessorsAddr[0], "Node.GetSuccessorListRPC", struct{}{}, &getSuccessorListRPCReply)
	successorListReply := getSuccessorListRPCReply.SuccessorList
	if err == nil {
		for i := 0; i < len(successorListReply)-1; i++ {
			node.SuccessorsAddr[i+1] = successorListReply[i]

		}
	} else {
		log.Println("Failed to get successor list", err)
		if node.SuccessorsAddr[0] == "" {
			log.Println("successorList[0] is empty, use itself as successorList[0]")
			node.SuccessorsAddr[0] = node.Addr
		} else {
			//successorList[0] is dead, remove it and shift the list to the upper
			for i := 0; i < len(node.SuccessorsAddr); i++ {
				if i == len(node.SuccessorsAddr)-1 {
					node.SuccessorsAddr[i] = ""
				} else {
					node.SuccessorsAddr[i] = node.SuccessorsAddr[i+1]
				}
			}

		}
	}

	//find the predecessor of the node's successor
	var getPredecessorRPCReply GetPredecessorRPCReply
	err = ChordCall(node.SuccessorsAddr[0], "Node.GetPredecessorRPC", struct{}{}, &getPredecessorRPCReply)
	//if the predecessor is not the active node
	//change the node's successor to the newer predecessor of pre-successor of the active node and notify
	if err == nil {
		var getSuccessorIDRPCReply GetIDRPCReply
		err = ChordCall(node.SuccessorsAddr[0], "Node.GetIDRPC", "", &getSuccessorIDRPCReply)
		if err != nil {
			log.Println("Failed to get successor[0] id")
			return err
		}
		successorID := getSuccessorIDRPCReply.Identifier

		predecessorAddr := getPredecessorRPCReply.PredecessorAddr
		var getPredecessorIDRPCReply GetIDRPCReply
		err = ChordCall(predecessorAddr, "Node.GETIDRPC", "", &getPredecessorIDRPCReply)
		if err != nil {
			log.Println("Failed to get predecessor id")
			return err
		}
		predecessorID := getPredecessorIDRPCReply.Identifier
		if predecessorAddr != "" && between(node.Identifier, predecessorID, successorID, false) {
			node.SuccessorsAddr[0] = predecessorAddr
		}

	}
	//notify
	ChordCall(node.SuccessorsAddr[0], "Node.NotifyRPC", node.Addr, &NotifyRPCReply{})
	//todo:copy node bucket

	return nil
}

// input the identifier of the node, 0-63
// return the start of the interval
func (node *Node) FingerStart(nodeId int) *big.Int {
	id := node.Identifier
	id = new(big.Int).Add(id, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(nodeId)-1), nil))
	return new(big.Int).Mod(id, hashMod)
}

func (node *Node) FixFingers() error {
	log.Println("--------------invocation of fixfingers function------------")

	node.nextFinger += 1
	if node.nextFinger > m {
		node.nextFinger = 1
	}
	// n + 2^next-1, this key is a file id
	key := new(big.Int).Add(node.Identifier, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(node.nextFinger)-1), nil))
	key.Mod(key, hashMod)
	// find the successor of the key
	next := Lookup(key, node.Addr)

	node.FingerTable[node.nextFinger].Addr = next
	node.FingerTable[node.nextFinger].Identifier = key.Bytes()
	return nil
}
