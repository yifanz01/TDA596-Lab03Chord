package main

import (
	"log"
	"math/big"
)

type LookupReply struct {
	Found         bool
	SuccessorAddr string
}

func Lookup(id *big.Int, startNode string) string {
	log.Println("---------------Invocation of Lookup start------------------")
	id.Mod(id, hashMod)
	next := startNode
	flag := false
	reply := LookupReply{}
	if !flag {
		ChordCall(next, "Node.FindSuccessorRPC", id, &reply)
		flag = reply.Found
		next = reply.SuccessorAddr
	}
	return next
}

/*
FindSuccessorRPCReply function reply and its implementation
*/

type FindSuccessorRPCReply struct {
	Found            bool
	SuccessorAddress string
}

type GetAddrRPCReply struct {
	Addr string
}

func (node *Node) FindSuccessorRPC(id *big.Int, reply *FindSuccessorRPCReply) {
	log.Println("---------------invocation of FindSuccessor----------------")
	var successorAddr string
	getAddrRPCReply := GetAddrRPCReply{}
	ChordCall(node.SuccessorsAddr[0], "Node.GetAddrRPC", "", getAddrRPCReply)

	successorAddr = getAddrRPCReply.Addr
	successorId := StrHash(successorAddr)
	successorId.Mod(successorId, hashMod)
	id.Mod(id, hashMod)

	flag := between(node.Identifier, id, successorId, true)

	reply.Found = false
	if flag {
		// the id is between node and its successor
		reply.Found = true
		reply.SuccessorAddress = node.SuccessorsAddr[0]
	} else {
		// find the successor from fingertable
		successorAddr = node.LookupFingerTable(id)
		findSuccessorRPCReply := FindSuccessorRPCReply{}
		ChordCall(successorAddr, "Node.FindSuccessorRPC", id, &findSuccessorRPCReply)
		reply.Found = findSuccessorRPCReply.Found
		reply.SuccessorAddress = findSuccessorRPCReply.SuccessorAddress
	}
}

func (node *Node) GetAddrRPC(reply GetAddrRPCReply) {
	reply.Addr = node.Name

}

func (node *Node) LookupFingerTable(id *big.Int) string {
	log.Println("--------------invocation of LookupFingerTable--------------")
	size := len(node.FingerTable)
	for i := size - 1; i >= 1; i-- {
		getAddrRPCReply := GetAddrRPCReply{}
		ChordCall(node.FingerTable[i].Addr, "Node.GetAddrRPC", "", &getAddrRPCReply)

		fingerId := StrHash(getAddrRPCReply.Addr)
		fingerId.Mod(fingerId, hashMod)
		flag := between(node.Identifier, fingerId, id, true)
		if flag {
			return node.FingerTable[i].Addr
		}
	}
	return node.SuccessorsAddr[0]
}
