package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/rpc/jsonrpc"
	"os"
	"strings"
)

func (node *Node) stabilize() error {
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

	//change the active node's successor to the successor[0]'s predecessor
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
		err = ChordCall(predecessorAddr, "Node.GetIDRPC", "", &getPredecessorIDRPCReply)
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
	err = ChordCall(node.SuccessorsAddr[0], "Node.NotifyRPC", node.Addr, &NotifyRPCReply{})
	if err != nil {
		log.Printf("[stabilize] Notify rpc error: %s\n", err)
	}
	// 1. First delete successor's backup
	// 2. Copy current bucket to successor's backup(do not do it if there is one node left)
	deleteSuccessorBackupRPCReply := DeleteSuccessorBackupRPCReply{}
	err = ChordCall(node.SuccessorsAddr[0], "Node.DeleteSuccessorBackupRPC", struct{}{}, &deleteSuccessorBackupRPCReply)
	if err != nil {
		log.Println("Delete successor's backup error: ", err)
		return err
	}

	if node.SuccessorsAddr[0] == node.Addr {
		return nil
	}
	for key, value := range node.Bucket {
		newFile := FileStructure{}
		newFile.Id = key
		newFile.Name = value
		filePath := "../files/" + "N" + node.Identifier.String() + "/chord_storage/" + value
		file, err := os.Open(filePath)
		if err != nil {
			log.Println("Open node's bucket file error: ", err)
			return err
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			log.Println("Read node's bucket file error: ", err)
			return err
		}
		//encrypt the file
		newFile.Content = content
		//encrypt the content
		var getPublicKeyRPCReply GetPublicKeyRPCReply
		err = ChordCall(node.SuccessorsAddr[0], "Node.GetPublicKeyRPC", "", &getPublicKeyRPCReply)
		if node.EncryptFlag {
			newFile.Content, _ = rsa.EncryptPKCS1v15(rand.Reader, getPublicKeyRPCReply.Public_Key, newFile.Content)
		}

		successorStoreFileReply := SuccessorStoreFileRPCReply{}
		err = ChordCall(node.SuccessorsAddr[0], "Node.SuccessorStoreFileRPC", newFile, &successorStoreFileReply)
		if successorStoreFileReply.Error != nil || err != nil {
			log.Println("[stabilize] Store files to successor error: ", successorStoreFileReply.Error, " and: ", err)
			return nil
		}

	}
	// Clean the redundant file in successor's backup
	node.cleanRedundantFile()

	return nil
}

func (node *Node) cleanRedundantFile() {
	// Read all local storage files
	filePath := "../files/" + "N" + node.Identifier.String() + "/chord_storage"
	files, err := os.ReadDir(filePath)
	if err != nil {
		log.Println("[cleanRedundantFile] Read directory error: ", err)
		return
	}
	for _, file := range files {
		fileName := file.Name()
		fileId := StrHash(fileName)
		fileId.Mod(fileId, hashMod)

		inBucket := false
		inBackup := false
		for id, _ := range node.Bucket {
			if id.Cmp(fileId) == 0 {
				inBucket = true
			}
		}

		for id, _ := range node.Backup {
			if id.Cmp(fileId) == 0 {
				inBackup = true
			}
		}

		if !inBackup && !inBucket {
			// The file is not in backup and bucket, delete it
			path := filePath + "/" + fileName
			err = os.Remove(path)
			if err != nil {
				log.Printf("[cleanRedundantFile] Cannot remove the file[%s]: ", path)
				return
			}
		}
	}
}

// input the identifier of the node, 0-63
// return the start of the interval
func (node *Node) FingerStart(nodeId int) *big.Int {
	id := node.Identifier
	id = new(big.Int).Add(id, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(nodeId)-1), nil))
	return new(big.Int).Mod(id, hashMod)
}

// FixFingers updates finger table
func (node *Node) FixFingers() error {

	node.nextFinger += 1
	if node.nextFinger > m {
		node.nextFinger = 1
	}
	// n + 2^next-1, this key is a file id
	//todo:why not using node.FingerTable[node.nextFinger].Identifier
	//
	key := new(big.Int).Add(node.Identifier, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(node.nextFinger)-1), nil))
	key.Mod(key, hashMod)

	// find the successor of the key
	next := Lookup(key, node.Addr)

	node.FingerTable[node.nextFinger].Addr = next
	node.FingerTable[node.nextFinger].Identifier = key.Bytes()

	// optimization,
	//for {
	//	node.nextFinger += 1
	//	if node.nextFinger > m {
	//		node.nextFinger = 0
	//	}
	//	key = new(big.Int).Add(node.Identifier, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(node.nextFinger)-1), nil))
	//	key.Mod(key, hashMod)
	//
	//	next = Lookup(key, node.Addr)
	//	successorId := StrHash(next)
	//	successorId.Mod(successorId, hashMod)
	//	if between(node.Identifier, key, successorId, false) {
	//		if node.FingerTable[node.nextFinger].Addr != next {
	//			node.FingerTable[node.nextFinger].Addr = next
	//			node.FingerTable[node.nextFinger].Identifier = key.Bytes()
	//		}
	//	} else {
	//		node.nextFinger -= 1
	//		return nil
	//	}
	//}
	return nil
}

// check whether predecessor has failed
func (node *Node) checkPredecessor() error {
	pred := node.PredecessorAddr
	if pred != "" {
		ip := strings.Split(pred, ":")[0]
		port := strings.Split(pred, ":")[1]

		// ip = NAT(ip)

		predAddr := ip + ":" + port
		_, err := jsonrpc.Dial("tcp", predAddr)
		if err != nil {
			fmt.Printf("Predecessor %s has failed\n", pred)
			node.PredecessorAddr = ""
			for k, v := range node.Backup {
				if v != "" {
					node.Bucket[k] = v
				}
			}

		}
	}
	return nil
}
