package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
// var _ = net.Listen
// var _ = os.Exit

func main() {
	fmt.Println("Starting the server at port 4221.")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221.")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	var data = make([]byte, 1024)

	totalBytesRead, err := conn.Read(data)
	if err != nil {
		fmt.Println("Error reading data.", err.Error())
		os.Exit(1)
	}

	request := string(data)
	fmt.Println("Total bytes read: ", totalBytesRead)
	fmt.Println("`request` size (in bytes): ", len(request))
	fmt.Println(request)

	lines := strings.Split(request, "\r\n")

	if len(lines) > 0 {
		firstHeader := lines[0]
		fmt.Println(firstHeader)
		res := strings.Split(firstHeader, " ")
		path := res[1]
		fmt.Println(path)
		if path == "/" {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		} else if strings.Contains(path, "echo") {
			responseData := strings.Split(path, "/")[2]
			response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(responseData), responseData)
			if err != nil {
				fmt.Println("Error formatting data.", err.Error())
				os.Exit(1)
			}
			conn.Write([]byte(response))
		} else {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}

	}

	// conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	// fmt.Println("No data sent. Closing connection.")
	// conn.Close()
}
