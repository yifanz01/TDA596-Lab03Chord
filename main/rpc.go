package main

import (
	"errors"
	"log"
	"net/rpc/jsonrpc"
	"strings"
)

/*
targetNodeAddr: connect object
serviceMethod: targetNodeAddr.serviceMethod and return reply
args: arguments for serviceMethod
reply: reply from serviceMethod
*/
func ChordCall(targetNodeAddr string, serviceMethod string, args interface{}, reply interface{}) error {
	if len(strings.Split(targetNodeAddr, ":")) != 2 {
		log.Println("Node ip:port address error!", targetNodeAddr)
		return errors.New("Error: targetNode address is not in the correct format: " + string(targetNodeAddr))
	}

	conn, err := jsonrpc.Dial("tcp", targetNodeAddr)
	if err != nil {
		log.Println("Method: ", serviceMethod, "dial error: ", err)
		return err
	}
	defer conn.Close()
	// serviceMethod's error will pass to err
	err = conn.Call(serviceMethod, args, reply)
	if err != nil {
		log.Println("Call error: ", err)
		return err
	}
	return nil
}
