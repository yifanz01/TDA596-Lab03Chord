package main

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
)

type GetIDRPCReply struct {
	Identifier *big.Int
}

func (node *Node) GetIDRPC(none string, reply *GetIDRPCReply) error {
	reply.Identifier = node.Identifier
	return nil
}

type NotifyRPCReply struct {
	Success bool
}

// change the predecessor of the node to addr
func (node *Node) notify(addr string) (bool, error) {
	if node.PredecessorAddr != "" {
		var getPredecessorIDRPCReply GetIDRPCReply
		err := ChordCall(node.PredecessorAddr, "Node.GetIDRPC", "", &getPredecessorIDRPCReply)
		if err != nil {
			log.Println("Failed to get the name of predecessor:", err)
			return false, err
		}
		predecessorID := getPredecessorIDRPCReply.Identifier

		var getAddrIDRPCReply GetIDRPCReply
		err = ChordCall(addr, "Node.GetIDRPC", "", &getAddrIDRPCReply)
		if err != nil {
			log.Println("Failed to get the name of predecessor:", err)
			return false, err
		}
		addrID := getAddrIDRPCReply.Identifier

		if between(predecessorID, addrID, node.Identifier, false) {
			node.PredecessorAddr = addr
			// log.Println(node.Name, "'s Predecessor is set to ", addr)
			return true, nil
		} else {
			return false, nil
		}
	} else {
		node.PredecessorAddr = addr
		// log.Println(node.Name, "'s Predecessor is set to ", addr)
		return true, nil
	}
}

func (node *Node) NotifyRPC(addr string, reply *NotifyRPCReply) error {
	//todo:move files
	if node.SuccessorsAddr[0] != node.Addr {
		node.moveFiles(addr)
	}

	reply.Success, _ = node.notify(addr)
	return nil
}

type GetSuccessorListRPCReply struct {
	SuccessorList []string
}

func (node *Node) GetSuccessorListRPC(none *struct{}, reply *GetSuccessorListRPCReply) error {
	reply.SuccessorList = node.SuccessorsAddr
	return nil
}

type GetPredecessorRPCReply struct {
	PredecessorAddr string
}

func (node *Node) GetPredecessorRPC(none *struct{}, reply *GetPredecessorRPCReply) error {
	reply.PredecessorAddr = node.PredecessorAddr
	if reply.PredecessorAddr == "" {
		return errors.New("predecessor is empty")
	} else {
		return nil
	}
}

type LookupReply struct {
	Found         bool
	SuccessorAddr string
}

func Lookup(id *big.Int, startNode string) string {
	//log.Println("---------------Invocation of Lookup start------------------")
	id.Mod(id, hashMod)
	next := startNode
	flag := false
	result := FindSuccessorRPCReply{}
	if !flag {
		err := ChordCall(next, "Node.FindSuccessorRPC", id, &result)
		if err != nil {
			log.Printf("[Lookup] Find successor rpc error: %s\n", err)
		}
		flag = result.Found
		next = result.SuccessorAddress
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

func (node *Node) FindSuccessorRPC(id *big.Int, reply *FindSuccessorRPCReply) error {
	//log.Println("---------------invocation of FindSuccessor----------------")
	var successorAddr string
	getAddrRPCReply := GetAddrRPCReply{}
	err := ChordCall(node.SuccessorsAddr[0], "Node.GetAddrRPC", "", &getAddrRPCReply)
	if err != nil {
		log.Printf("[FindSuccessorRPC] Get AddrRPC error: %s\n", err)
	}

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
		err = ChordCall(successorAddr, "Node.FindSuccessorRPC", id, &findSuccessorRPCReply)
		if err != nil {
			log.Printf("[FindSuccessorRPC] Find successor rpc error: %s", err)
		}
		reply.Found = findSuccessorRPCReply.Found
		reply.SuccessorAddress = findSuccessorRPCReply.SuccessorAddress
	}
	return nil
}

func (node *Node) GetAddrRPC(none string, reply *GetAddrRPCReply) error {
	reply.Addr = node.Addr
	return nil
}

type SetPredecessorRPCReply struct {
	Success bool
}

func (node *Node) SetPredecessorRPC(predecessorAddr string, reply *SetPredecessorRPCReply) error {
	//fmt.Println("-------------- Invoke SetPredecessorRPC function ------------")
	node.PredecessorAddr = predecessorAddr
	reply.Success = true
	return nil
}

func (node *Node) LookupFingerTable(id *big.Int) string {
	//log.Println("--------------invocation of LookupFingerTable--------------")
	size := len(node.FingerTable)
	for i := size - 1; i >= 1; i-- {
		getAddrRPCReply := GetAddrRPCReply{}
		err := ChordCall(node.FingerTable[i].Addr, "Node.GetAddrRPC", "", &getAddrRPCReply)
		if err != nil {
			log.Printf("[LookupFingerTable] Get addrRPC error: %s\n", err)
		}

		fingerId := StrHash(getAddrRPCReply.Addr)
		fingerId.Mod(fingerId, hashMod)
		flag := between(node.Identifier, fingerId, id, false) // Todo, why is false. e.g. 42  (8, 54]
		if flag {
			return node.FingerTable[i].Addr
		}
	}
	return node.SuccessorsAddr[0]
}

type GetPublicKeyRPCReply struct {
	Public_Key *rsa.PublicKey
}

func (node *Node) GetPublicKeyRPC(none string, reply *GetPublicKeyRPCReply) error {
	reply.Public_Key = node.PublicKey
	return nil
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
	filePath := "../files/" + "N" + node.Identifier.String() + "/upload/"
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

	//encrypt the file
	var getPublicKeyRPCReply GetPublicKeyRPCReply
	err = ChordCall(addr, "Node.GetPublicKeyRPC", "", &getPublicKeyRPCReply)
	if node.EncryptFlag {
		newFile.Content, _ = rsa.EncryptPKCS1v15(rand.Reader, getPublicKeyRPCReply.Public_Key, newFile.Content)
	}

	// send storefile rpc
	reply := StoreFileRPCReply{}
	reply.Backup = false
	err = ChordCall(addr, "Node.StoreFileRPC", newFile, &reply)

	return err
}

func (node *Node) StoreFileRPC(f FileStructure, reply *StoreFileRPCReply) error {
	//log.Println("-----------------invocation of StoreFileRPC start------------------")

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

	filePath := "../files/" + "N" + node.Identifier.String() + "/chord_storage/" + f.Name

	file, err := os.Create(filePath)
	if err != nil {
		log.Println("Create file error: ", err)
		return false
	}
	defer file.Close()

	//before writing, decrypt the file
	f.Content, err = rsa.DecryptPKCS1v15(rand.Reader, node.PrivateKey, f.Content)

	if err != nil {
		log.Println("Failed to decrypt the file ", err)
	}
	_, err = file.Write(f.Content)
	if err != nil {
		log.Println("Write file error: ", err)
		return false
	}
	return true
}

type CheckFileExistRPCReply struct {
	Exist bool
}

func (node *Node) CheckFileExistRPC(fileName string, reply *CheckFileExistRPCReply) error {
	//log.Println("----------------invocation of checkfileexistRPC---------------")
	for _, value := range node.Bucket {
		if fileName == value {
			reply.Exist = true
			return nil
		}
	}
	reply.Exist = false
	return nil
}

type DeleteSuccessorBackupRPCReply struct {
	Success bool
}

func (node *Node) DeleteSuccessorBackupRPC(args interface{}, reply *DeleteSuccessorBackupRPCReply) error {
	reply.Success = node.deleteSuccessorBackupRPC()
	return nil
}

func (node *Node) deleteSuccessorBackupRPC() bool {
	for key, _ := range node.Backup {
		// we just remove the reference to the key, but the file still exists in local disk. It will be cleaned later
		delete(node.Backup, key)
	}
	return true
}

type SuccessorStoreFileRPCReply struct {
	Successor bool
	Error     error
}

func (node *Node) SuccessorStoreFileRPC(f FileStructure, reply *SuccessorStoreFileRPCReply) error {
	reply.Successor = node.successorStoreFile(f)
	if !reply.Successor {
		reply.Error = errors.New("SuccessorStoreFileRPC error!")
		return reply.Error
	} else {
		reply.Error = nil
		return nil
	}
}

func (node *Node) successorStoreFile(f FileStructure) bool {
	f.Id.Mod(f.Id, hashMod)
	for k, _ := range node.Backup {
		if k.Cmp(f.Id) == 0 {
			return true
		}
	}
	for k, _ := range node.Bucket {
		if k.Cmp(f.Id) == 0 {
			return true
		}
	}
	node.Backup[f.Id] = f.Name
	filePath := "../files/" + "N" + node.Identifier.String() + "/chord_storage/" + f.Name
	file, err := os.Create(filePath)
	if err != nil {
		log.Println("[successorStoreFile] Create file error: ", err)
		return false
	}
	defer file.Close()
	_, err = file.Write(f.Content)
	if err != nil {
		log.Println("[successorStoreFile] File write error: ", err)
		return false
	}
	//log.Printf("[successorStoreFile] File:%s store success!\n", f.Name)
	return true
}

func (node *Node) moveFiles(addr string) {
	var getIdReply GetIDRPCReply
	err := ChordCall(addr, "Node.GetIDRPC", "", &getIdReply)
	if err != nil {
		log.Println("Failed to get the id:", err)
	}
	addrId := getIdReply.Identifier
	addrId.Mod(addrId, hashMod)

	// iterate local bucket
	for k, v := range node.Bucket {
		fileId := k
		fileName := v
		filePath := "../files/" + "N" + node.Identifier.String() + "/chord_storage/" + fileName
		file, err := os.Create(filePath)
		if err != nil {
			log.Println("[moveFiles] File cannot be open: ", err)
			return
		}
		defer file.Close()
		newFile := FileStructure{}
		newFile.Name = fileName
		newFile.Id = fileId
		newFile.Content, err = io.ReadAll(file)

		//encrypt the file
		var getPublicKeyRPCReply GetPublicKeyRPCReply
		err = ChordCall(addr, "Node.GetPublicKeyRPC", "", &getPublicKeyRPCReply)
		if node.EncryptFlag {
			newFile.Content, _ = rsa.EncryptPKCS1v15(rand.Reader, getPublicKeyRPCReply.Public_Key, newFile.Content)
		}

		if err != nil {
			log.Println("[moveFiles] File cannot be read: ", err)
			return
		}
		if between(fileId, addrId, node.Identifier, true) {
			var moveFileReply StoreFileRPCReply
			moveFileReply.Backup = false
			err = ChordCall(addr, "Node.StoreFileRPC", newFile, &moveFileReply)
			if err != nil {
				log.Println("[moveFiles] Move file error: ", err)
			}
			// delete local file
			delete(node.Bucket, k)
		}
	}
}
