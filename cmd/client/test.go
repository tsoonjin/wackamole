package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	fmt.Println("Welcome to interactive console")
	// create channel to send data from stdin
	input := make(chan string)
	// start goroutine to keep listen what is typed to console input (stdin)
	go func(in chan string) {
		// create new reader from stdin
		reader := bufio.NewReader(os.Stdin)
		// start infinite loop to continuously listen to input
		for {
			// read by one line (enter pressed)
			s, err := reader.ReadString('\n')
			// check for errors
			if err != nil {
				// close channel just to inform others
				close(in)
				log.Println("Error in read string", err)
			}
			in <- s
		}
		// pass input channel to closure func
	}(input)
	// label to jump from break in select
	// it will not go inside again
exit:
	// start infinite loop to reply input data
	for {
		// use select to wait until some data come to input channel
		select {
		case in := <-input:
			// remove all leading and trailing white space
			in = strings.TrimSpace(in)
			if in == "exit" {
				// if exit command received
				// break from infinite loop to label and go next
				// line after for loop
				break exit
			}
			// do what you want with input data
			fmt.Println("Read from stdin: ", in)
		}
	}

	// on exit be polite
	fmt.Println("Bye, have a nice day")
}
