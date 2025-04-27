package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func handleRequest(conn net.Conn) {
	defer conn.Close()
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
		res := strings.Fields(firstHeader)
		path := res[1]
		fmt.Println("path: ", path)
		// for idx, line := range lines {
		// 	fmt.Printf("line %d: \n--------------------\n%s\n--------------------\n", idx, line)
		// }

		// Handle root path
		if path == "/" {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		} else if strings.Contains(path, "echo") {

			responseData := strings.Split(path, "/")[2]
			response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(responseData), responseData)
			conn.Write([]byte(response))

		} else if strings.Contains(path, "user-agent") {

			var userAgent string
			for _, line := range lines {
				if strings.HasPrefix(line, "User-Agent:") {

					userAgent = strings.TrimSpace(strings.TrimPrefix(line, "User-Agent:"))
					break
				}
			}

			if userAgent != "" {
				response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)
				conn.Write([]byte(response))
			} else {
				conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			}

		} else {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
	}

}

func main() {
	fmt.Println("Starting the server at port 4221.")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221.")
		os.Exit(1)
	}
	defer l.Close()

	for {
		// NOTE: this part is synchronous and waits for connections.
		// It's what allows the loop to wait until further notice.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting a connection: ", err.Error())
			// NOTE: this is what allows us to continue accepting other connections.
			continue
		}

		// handle the request asynchronously.
		go handleRequest(conn)

	}

}
