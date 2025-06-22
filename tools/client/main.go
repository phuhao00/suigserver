package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	var host = flag.String("host", "localhost", "Server host")
	var port = flag.Int("port", 8080, "Server port")
	flag.Parse()

	// Connect to server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s:%d\n", *host, *port)
	fmt.Println("Commands: /auth <name>, /join <room>, /say <message>, /quit")
	fmt.Println("Type 'exit' to quit the client")

	// Start a goroutine to read from server
	go func() {
		reader := bufio.NewReader(conn)
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Connection lost: %v\n", err)
				os.Exit(1)
			}
			fmt.Print("Server: " + message)
		}
	}()

	// Read input from user and send to server
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "exit" {
			break
		}

		if input == "" {
			continue
		}

		_, err := conn.Write([]byte(input + "\n"))
		if err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
			break
		}
	}

	fmt.Println("Goodbye!")
}
