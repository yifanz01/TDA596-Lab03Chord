package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
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

type SetPredecessorRPCReply struct {
	Success bool
}

func (node *Node) SetPredecessorRPC(predecessorAddr string, reply *SetPredecessorRPCReply) {
	fmt.Println("-------------- Invoke SetPredecessorRPC function ------------")
	node.PredecessorAddr = predecessorAddr
	reply.Success = true
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

func (node *Node) PrintState() {
	fmt.Println("-------------- Current Node State ------------")
	fmt.Println("Node Name: ", node.Name)
	fmt.Println("Node Address: ", node.Addr)
	fmt.Println("Node Identifier: ", new(big.Int).SetBytes(node.Identifier.Bytes()))
	fmt.Println("Node Predecessor: ", node.PredecessorAddr)
	fmt.Println("Node Successors: ")
	for i := 0; i < len(node.SuccessorsAddr); i++ {
		fmt.Println("Successor ", i, " address: ", node.SuccessorsAddr[i])
	}
	fmt.Println("Node Finger Table: ")
	for i := 1; i < fingerTableLen+1; i++ {
		item := node.FingerTable[i]
		id := new(big.Int).SetBytes(item.Identifier)
		address := item.Addr
		fmt.Println("Finger ", i, " id: ", id, ", address: ", address)
		//todo:print bucket and backup

	}
}

type FileStructure struct {
	Id      *big.Int
	Name    string // file name e.g. "../files/" + node.Name + "/upload/"
	Content []byte
}

func StoreFile(fileName string, node *Node) error {
	// find which node should this file stored
	key := StrHash(fileName)
	addr := Lookup(key, node.Addr)
	// read the file and upload it to addr
	filePath := "../files/" + node.Name + "/upload/"
	filePath += fileName
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("The file cannot be opened!")
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Println("The file cannot be read!")
	}
	defer file.Close()

	newFile := FileStructure{}
	newFile.Name = fileName
	newFile.Id = key
	newFile.Id.Mod(newFile.Id, hashMod)
	newFile.Content = content

	// Todo encrypt file content

	// send storefile rpc
	reply := StoreFileRPCReply{}
	reply.Backup = false
	err = ChordCall(addr, "Node.StoreFileRPC", newFile, &reply)

	return err
}

func (node *Node) StoreFileRPC(f FileStructure, reply *StoreFileRPCReply) error {
	log.Println("-----------------invocation of StoreFileRPC start------------------")

	flag := node.storeFile(f, reply.Backup)
	reply.Success = flag
	if flag {
		log.Println("File storage success!")
	} else {
		log.Println("File storage error!")
		return errors.New("File storage error!")
	}
	return nil
}

func (node *Node) storeFile(f FileStructure, backUp bool) bool {
	// Store the file in the bucket
	// Return true if success, false if failed
	// Append the file to the bucket

	// check if file is already in the bucket
	if backUp {
		for k, _ := range node.Backup {
			if k.Cmp(f.Id) == 0 {
				log.Println("This file already exists Backup")
				return false
			}
		}
		node.Backup[f.Id] = f.Name
		fmt.Println("Store Backup: ", node.Backup)
	} else {
		for k, _ := range node.Bucket {
			if k.Cmp(f.Id) == 0 {
				log.Println("This file already exists in Bucket")
				return false
			}
		}
		node.Bucket[f.Id] = f.Name
		fmt.Println("Store Bucket: ", node.Bucket)
	}

	filePath := "../files/" + node.Name + "/storage/" + f.Name

	file, err := os.Create(filePath)
	if err != nil {
		log.Println("Create file error: ", err)
		return false
	}
	defer file.Close()

	_, err = file.Write(f.Content)
	if err != nil {
		log.Println("Write file error: ", err)
		return false
	}
	return true
}
