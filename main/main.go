package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strings"
	"time"
)

type ScheduledExecutor struct {
	delay  time.Duration
	ticker time.Ticker
	quit   chan int
}

func (s *ScheduledExecutor) Start(task func()) {
	s.ticker = *time.NewTicker(s.delay)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				go task()
			case <-s.quit:
				s.ticker.Stop()
				return
			}
		}
	}()
}

func main() {
	arguments := getComArgs()
	log.Println("Arguments: ", arguments)

	flag := validArguments(arguments)
	if flag == -1 {
		log.Println("[main] Arguments are invalid!")
		os.Exit(1)
	} else {
		log.Println("[main] Arguments are valid!")
		node := NewNode(arguments)

		IpAddress := fmt.Sprintf("%s:%d", arguments.IpAddress, arguments.Port)
		addr, err := net.ResolveTCPAddr("tcp", IpAddress)
		if err != nil {
			log.Fatalln("ResolveTCPAddr failed:", err.Error())
		}
		rpc.Register(node)
		listener, err := net.Listen("tcp", addr.String())
		if err != nil {
			log.Fatalln("ListenTCP failed:", err.Error())
		}
		defer listener.Close()

		go func(listener net.Listener) {
			for {
				conn, err := listener.Accept()
				if err != nil {
					fmt.Println("Accept failed:", err.Error())
					continue
				}
				go jsonrpc.ServeConn(conn)
			}
		}(listener)

		if flag == 0 {
			// Join the existing chord
			remoteAddress := fmt.Sprintf("%s:%d", arguments.JoinAddress, arguments.JoinPort)
			node.joinChord(remoteAddress)
		} else if flag == 1 {
			// Create new chord
			node.createNewChord()
		}

		executorStabilization := ScheduledExecutor{
			delay: time.Duration(arguments.Ts) * time.Millisecond,
			quit:  make(chan int),
		}
		executorStabilization.Start(func() {
			node.stabilize()
		})

		executorFixFinger := ScheduledExecutor{
			delay: time.Duration(arguments.Tff) * time.Millisecond,
			quit:  make(chan int),
		}
		executorFixFinger.Start(func() {
			node.FixFingers()
		})

		executorCheckPredecessor := ScheduledExecutor{
			delay: time.Duration(arguments.Tcp) * time.Millisecond,
			quit:  make(chan int),
		}
		executorCheckPredecessor.Start(func() {
			node.checkPredecessor()
		})

		// Read input from stdin
		reader := bufio.NewReader(os.Stdin)
		for {
			log.Println("Please enter your command(Lookup/StoreFile/PrintState/Quit)...")
			command, _ := reader.ReadString('\n')
			command = strings.ToUpper(strings.TrimSpace(command))
			if command == "LOOKUP" {
				log.Println("Please enter the file you want to look up...")
				fileName, _ := reader.ReadString('\n')
				fileName = strings.TrimSpace(fileName)
				// hash this fila name to m-digits number
				key := StrHash(fileName)
				targetAddr := Lookup(key, node.Addr)
				log.Println("The node that could has the required data: ", targetAddr)

				// check if the file exists in targetAddr
				checkFileExistRPCReply := CheckFileExistRPCReply{}
				err = ChordCall(targetAddr, "Node.CheckFileExistRPC", fileName, &checkFileExistRPCReply)
				if err != nil {
					log.Println("Check file exist fail..", err)
					continue
				} else {
					if checkFileExistRPCReply.Exist {
						var getAddrRPCReply GetAddrRPCReply
						err = ChordCall(targetAddr, "Node.GetAddrRPC", "", &getAddrRPCReply)
						if err != nil {
							log.Println("Chord Call failed! ")
							continue
						} else {
							log.Println("The file is stored at: ", targetAddr)
						}
					} else {
						log.Println("The file is not stored at this node: ", targetAddr)
					}
				}
			} else if command == "STOREFILE" {
				log.Println("Please enter the file you want to upload...")
				fileName, _ := reader.ReadString('\n')
				fileName = strings.TrimSpace(fileName)
				err = StoreFile(fileName, node)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("File storage success!")
				}

			} else if command == "PRINTSTATE" {
				node.PrintState()
			} else if command == "QUIT" {
				executorStabilization.quit <- 1
				executorFixFinger.quit <- 1
				executorCheckPredecessor.quit <- 1
				os.Exit(0)
			} else {
				log.Println("Invalid command! Please enter your command again(Lookup/StoreFile/PrintState/Quit)...")
			}
		}
	}
}
