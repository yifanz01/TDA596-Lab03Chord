package main

import (
	"log"
	"net/rpc/jsonrpc"
	"strings"
)

func ChordCall(targetNodeAddr string, serviceMethod string, args interface{}, reply interface{}) {
	if len(strings.Split(targetNodeAddr, ":")) != 2 {
		log.Fatalln("Node ip:port address error!", targetNodeAddr)
	}

	conn, err := jsonrpc.Dial("tcp", targetNodeAddr)
	if err != nil {
		log.Fatalln("Method: ", serviceMethod, "dial error: ", err)
	}
	defer conn.Close()
	err = conn.Call(serviceMethod, args, reply)
	if err != nil {
		log.Fatalln("Call error: ", err)
	}
}
