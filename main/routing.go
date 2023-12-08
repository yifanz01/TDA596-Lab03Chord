package main

import (
	"log"
	"math/big"
)

type LookupReply struct {
	Found         bool
	SuccessorAddr string
}

func Lookup(id *big.Int, node string) string {
	log.Println("---------------Invocation of Lookuo start------------------")
	next := node
	var found bool = false
	reply := LookupReply{}
	if !found {
		ChordCall(next, "Node.FindSuccessorRPC", id, &reply)
		found = reply.Found
		next = reply.SuccessorAddr
	}
	return next
}
