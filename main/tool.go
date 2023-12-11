package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
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
	flag.Parse()

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
	if args.ClientName != "default" {
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
			// Join the chord ring
			return 0
		} else {
			log.Println("Joining address is invalid")
			return -1
		}
	} else {
		// Create a new chord ring
		// ignore jp input
		return 1
	}
}

// hash file name to m-digits number
func StrHash(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

func between(start, elt, end *big.Int, inclusive bool) bool {
	if end.Cmp(start) > 0 { // start < end
		return (start.Cmp(elt) < 0 && elt.Cmp(end) < 0) || (inclusive && elt.Cmp(end) == 0)
	} else {
		return start.Cmp(elt) < 0 || elt.Cmp(end) < 0 || (inclusive && elt.Cmp(end) == 0)
	}
}

type IP struct {
	Query string
}

func getLocalAddress() string {
	// get local ip address from dns server 8.8.8.8:80
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func (node *Node) genRSAKey(bits int) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		log.Println("[genRSAKey] Failed to generate private key for node ", node.Name, "N"+node.Identifier.String())
	}
	node.PrivateKey = privateKey
	node.PublicKey = &privateKey.PublicKey

	//store private key in the node folder
	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	block := pem.Block{Type: "N" + node.Identifier.String() + "-private Key",
		Headers: nil,
		Bytes:   privateKeyDER}
	nodeFolder := "../files/" + "N" + node.Identifier.String()
	privateKeyFile, err := os.Create(nodeFolder + "/private.pem")
	if err != nil {
		log.Println("[genRSAKey] Failed to create private key file for node ", node.Name, "N"+node.Identifier.String())
	}
	defer privateKeyFile.Close()
	err = pem.Encode(privateKeyFile, &block)
	if err != nil {
		log.Println("[genRSAKey] Failed to write private key into file")
	}

	//store public kay in the node folder
	publicKeyDER, err := x509.MarshalPKIXPublicKey(node.PublicKey)
	if err != nil {
		log.Println("[genRSAKey] Failed to get DER format of public key for node ", node.Name)
	}
	block = pem.Block{
		Type:    "N" + node.Identifier.String() + "-public Key",
		Headers: nil,
		Bytes:   publicKeyDER,
	}
	publicKeyFile, err := os.Create(nodeFolder + "/public.pem")
	if err != nil {
		log.Println("[genRSAKey] Failed to create public key file for node ", node.Name)
	}
	defer publicKeyFile.Close()
	err = pem.Encode(publicKeyFile, &block)
	if err != nil {
		log.Println("[genRSAKey] Failed to write public key into file")
	}
}

// EncryptFile Encrypt file
func (node *Node) EncryptFile(content []byte) []byte {
	publicKey := node.PublicKey
	encryptedContent, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, content)
	if err != nil {
		fmt.Println("Encrypt file failed")
		return nil
	}
	// Return the encrypted file
	return encryptedContent
}

// DecryptFile Decrypt file
func (node *Node) DecryptFile(content []byte) []byte {
	privateKey := node.PrivateKey
	decryptedContent, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, content)
	if err != nil {
		fmt.Println("Decrypt file failed")
		return decryptedContent
	}
	// Return the decrypted file
	return decryptedContent
}

func NAT(addr string) string {
	/*
	* NAT: ip is internal ip, need to be changed to external ip
	 */
	new_addr := addr
	getLocalAddress_res := getLocalAddress()
	// fmt.Println("getLocalAddress_res: ", getLocalAddress_res)
	// fmt.Println("Input addr: ", addr)
	if addr == getLocalAddress_res {
		new_addr = "localhost"
	}

	// wwq's NAT
	if addr == "172.31.21.112" {
		new_addr = "54.145.27.145"
	}

	// cfz's NAT
	if addr == "192.168.31.236" {
		new_addr = "95.80.36.91"
	}

	// jetson's NAT
	if addr == "192.168.31.153" {
		new_addr = "95.80.36.91"
	}
	// qi's laptop NAT
	if addr == "192.168.254.89" {
		new_addr = "50.93.222.140"
	}

	// qi's AWS NAT
	if addr == "172.31.82.96" {
		new_addr = "18.233.168.46"
	}

	return new_addr
}

func getIP() string {
	req, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		fmt.Println("could not get ip")
		return err.Error()
	}
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err.Error()
	}

	var ip IP
	fmt.Println("body: ", string(body))
	json.Unmarshal(body, &ip)

	return ip.Query
}
