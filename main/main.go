package main

import (
	"log"
	"os"
)

func main() {
	arguments := getComArgs()
	log.Println("Arguments: ", arguments)

	flag := validArguments(arguments)
	if flag == -1 {
		log.Println("Arguments are invalid!")
		os.Exit(-1)
	} else {

	}
}
