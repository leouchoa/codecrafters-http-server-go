package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func newHandleRequest(conn net.Conn, directoryPath string) {
	defer conn.Close()

	var data = make([]byte, 1024)
	totalBytesRead, err := conn.Read(data)
	if err != nil {
		fmt.Println("Error reading data:", err.Error())
		os.Exit(1)
	}

	// WARN: this is critical to only take the actual bytes read
	request := string(data[:totalBytesRead])
	fmt.Println("Total bytes read: ", totalBytesRead)
	fmt.Println("`request` size (in bytes): ", len(request))
	fmt.Println("Request:\n\n", request)

	lines := strings.Split(request, "\r\n")

	var contentLength int

	// NOTE: Parsing headers to extract the Content-Length
	// ----------------------------------------------------------------------
	// Extract Content-Length header from the request to determine how
	// much data we need to read from the connection. The Content-Length
	// specifies the number of bytes in the request body. By dynamically
	// allocating a buffer based on this value, we avoid unnecessary memory
	// usage and ensure that we read exactly the amount of data that was sent.
	// ----------------------------------------------------------------------
	for _, line := range lines {
		if strings.HasPrefix(line, "Content-Length:") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				contentLength, err = strconv.Atoi(parts[1])
				if err != nil {
					conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
					return
				}
			}
		}
	}

	if len(lines) > 0 {
		firstHeader := lines[0]
		fmt.Println(firstHeader)
		res := strings.Fields(firstHeader)
		path := res[1]

		// Handle the requests
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

		} else if strings.Contains(path, "files") {
			fmt.Println("Inside `files` path:", res)
			filename := strings.Split(path, "/")[2]
			filepath := filepath.Join(directoryPath, filename)

			if res[0] == "GET" {
				fileContent, err := readFile(filepath)
				if err != nil {
					conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
				}
				response := fmt.Sprintf(
					"HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s",
					len(fileContent),
					fileContent,
				)
				conn.Write([]byte(response))

			} else if res[0] == "POST" {
				// POST method received!!
				fmt.Println("POST method received!!")

				// Check if file exists
				fileExists := checkFileExists(filepath)
				if !fileExists {
					// If the file does not exist, we need to extract the payload and save it
					if contentLength > 0 {
						// Dynamically allocate a buffer based on Content-Length
						payload := make([]byte, contentLength)
						totalBytesRead, err := conn.Read(payload)
						if err != nil {
							fmt.Println("Error reading payload data:", err.Error())
							conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
							return
						}

						// Only read the amount of data specified by Content-Length
						payloadData := string(payload[:totalBytesRead])
						fmt.Println("Received payload:", payloadData)

						// Create the file with the payload data
						err = createFile(filepath, []byte(payloadData))
						if err != nil {
							conn.Write([]byte("HTTP/1.1 400 Error writing data\r\n\r\n"))
							return
						}

						conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
					} else {
						// If there's no Content-Length header or it's 0, return an error
						conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
					}

				} else {
					// If the file already exists
					conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
				}
			}

		} else {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
	}
}
