package main

import (
	"flag"
	"log"
	"net"
	"regexp"
)

type Arguments struct {
	IpAddress   string //The IP address that the Chord client will bind to.
	Port        int    //The port that the Chord client will bind to and listen on. Represented as a base-10 integer. Must be specified.
	JoinAddress string //The IP address of the machine running a Chord node
	JoinPort    int    //The port that an existing Chord node is bound to and listening on
	Ts          int    //The time in milliseconds between invocations of ‘stabilize’.
	Tff         int    //The time in milliseconds between invocations of ‘fix fingers’
	Tcp         int    //The time in milliseconds between invocations of ‘check predecessor’
	R           int    //The number of successors maintained by the Chord client.
	ClientName  string //The identifier (ID) assigned to the Chord client which will override the ID computed by the SHA1 sum of the client’s IP address and port number.
}

func getComArgs() Arguments {
	// Read command line arguments
	var a string  // Current node address
	var p int     // Current node port
	var ja string // Joining node address
	var jp int    // Joining node port
	var ts int    // The time in milliseconds between invocations of stabilize.
	var tff int   // The time in milliseconds between invocations of fix_fingers.
	var tcp int   // The time in milliseconds between invocations of check_predecessor.
	var r int     // The number of successors to maintain.
	var i string  // Client name

	flag.StringVar(&a, "a", "localhost", "current ip address")
	flag.IntVar(&p, "p", 8080, "current port")
	flag.StringVar(&ja, "ja", "Null", "joining node address")
	flag.IntVar(&jp, "jp", 8081, "joining node port")
	flag.IntVar(&ts, "ts", 3000, "the time in milliseconds between invocations of stabilize")
	flag.IntVar(&tff, "tff", 3000, "The time in milliseconds between invocations of fix_fingers.")
	flag.IntVar(&tcp, "tcp", 100, "The time in milliseconds between invocations of check_predecessor")
	flag.IntVar(&r, "r", 3, "The number of successors to maintain")
	flag.StringVar(&i, "i", "default", "Client name")

	return Arguments{
		IpAddress:   a,
		Port:        p,
		JoinAddress: ja,
		JoinPort:    jp,
		Ts:          ts,
		Tff:         tff,
		Tcp:         tcp,
		R:           r,
		ClientName:  i,
	}

}

func validArguments(args Arguments) int {
	if net.ParseIP(args.IpAddress) == nil && args.IpAddress != "localhost" {
		log.Println("Ip address is invalid!")
		return -1
	}
	if args.Port < 1024 || args.Port > 65535 {
		log.Println("Port number is invalid")
		return -1
	}
	// Check if durations are valid
	if args.Ts < 1 || args.Ts > 60000 {
		log.Println("Stabilize time is invalid")
		return -1
	}
	if args.Tff < 1 || args.Tff > 60000 {
		log.Println("FixFingers time is invalid")
		return -1
	}
	if args.Tcp < 1 || args.Tcp > 60000 {
		log.Println("CheckPred time is invalid")
		return -1
	}

	// Check if number of successors is valid
	if args.R < 1 || args.R > 32 {
		log.Println("Successors number is invalid")
		return -1
	}

	// Check if client name is s a valid string matching the regular expression [0-9a-fA-F]{40}
	if args.ClientName != "Default" {
		matched, err := regexp.MatchString("[0-9a-fA-F]*", args.ClientName)
		if err != nil || !matched {
			log.Println("Client Name is invalid")
			return -1
		}
	}

	// Check if joining address and port is valid or not
	if args.JoinAddress != "Null" {
		// Addr is specified, check if addr & port are valid
		if net.ParseIP(args.JoinAddress) != nil || args.JoinAddress == "localhost" {
			// Check if join port is valid
			if args.JoinPort < 1024 || args.JoinPort > 65535 {
				log.Println("Join port number is invalid")
				return -1
			}
			// Join the chord
			return 0
		} else {
			log.Println("Joining address is invalid")
			return -1
		}
	} else {
		// Join address is not specified, create a new chord ring
		// ignroe jp input
		return 1
	}
}
