package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func handleRequest(conn net.Conn, directoryPath string) {
	for {

		var data = make([]byte, 1024)

		totalBytesRead, err := conn.Read(data)
		if err != nil {
			if err.Error() == "EOF" {
				// NOTE: besides checking header you also need to search for EOF
				// If EOF is encountered, it simply means no more data for now,
				// so we continue to wait for more requests, not terminate.
				break
			}
			fmt.Println("Error reading data.", err.Error())
			os.Exit(1)
		}

		// WARN: this is critical to only take the actual bytes read
		request := string(data[:totalBytesRead])
		fmt.Println("Total bytes read: ", totalBytesRead)
		fmt.Println("`request` size (in bytes): ", len(request))
		fmt.Println("Request:\n\n", request)

		if checkCloseConnection(request) {
			conn.Write([]byte("HTTP/1.1 200 OK\r\nConnection: close\r\n\r\n"))
			conn.Close()
			break
		}

		lines := strings.Split(request, "\r\n")

		if len(lines) > 0 {
			firstHeader := lines[0]
			fmt.Println(firstHeader)
			res := strings.Fields(firstHeader)
			path := res[1]
			// fmt.Println("path: ", path)
			// for idx, line := range lines {
			// 	fmt.Printf("line %d: \n--------------------\n%s\n--------------------\n", idx, line)
			// }

			if path == "/" {
				conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			} else if strings.Contains(path, "echo") {

				var encodingType string
				for _, line := range lines {
					if strings.HasPrefix(line, "Accept-Encoding:") {

						encodingType = strings.TrimSpace(strings.TrimPrefix(line, "Accept-Encoding:"))
						break
					}
				}

				fmt.Println(encodingType)
				if encodingType == "" {
					responseData := strings.Split(path, "/")[2]
					response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(responseData), responseData)
					conn.Write([]byte(response))
				} else if strings.Contains(encodingType, "gzip") {
					responseData := strings.Split(path, "/")[2]
					compressedData, err := compressGzip(responseData)
					if err != nil {
						conn.Write([]byte(fmt.Sprintf("HTTP/1.1 500 Internal Server Error\r\n\r\n%s", err.Error())))
						return
					}

					// NOTE: Using `compressedData.Len()` because it's the length of
					// the already compressed data.
					compressedDataSize := compressedData.Len()
					response := fmt.Sprintf(
						"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Encoding: gzip\r\nContent-Length: %d\r\n\r\n",
						compressedDataSize)

					// NOTE: Write the header and then the compressed data
					conn.Write([]byte(response))
					conn.Write(compressedData.Bytes())
				} else {
					responseData := strings.Split(path, "/")[2]
					response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(responseData), responseData)
					conn.Write([]byte(response))
				}

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

				} else {
					// TODO:
					// 1. [X] check if file exists
					// 2. [X] create file
					// 3. [X] return response (error or 201/created)

					fmt.Println("POST method received!!")
					fileExists := checkFileExists(filepath)
					if !fileExists {

						payloadData, err := extractPayloadData(lines)
						if err != nil {
							conn.Write([]byte("HTTP/1.1 400 Error extracting payload\r\n\r\n"))
							return
						}
						err = createFile(filepath, []byte(payloadData))
						fmt.Println("extracted payload data: ", payloadData)
						if err != nil {
							conn.Write([]byte("HTTP/1.1 400 Error writing data\r\n\r\n"))
						}
						conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))

					} else {
						conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
					}

				}

			} else {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			}
		}
		if checkCloseConnection(request) {
			conn.Write([]byte("HTTP/1.1 200 OK\r\nConnection: close\r\n\r\n"))
			conn.Close()
			break
		}
	}

	conn.Close()
}

func checkCloseConnection(request string) bool {

	lines := strings.Split(request, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, "Connection: close") {
			return true
		}
	}
	return false

}

func compressGzip(data string) (bytes.Buffer, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	_, err := zw.Write([]byte(data))
	if err != nil {
		return bytes.Buffer{}, err
	}

	if err := zw.Close(); err != nil {
		return bytes.Buffer{}, err
	}

	return buf, nil
}

func extractPayloadData(lines []string) (string, error) {
	// last line is the payload data
	lastLine := lines[len(lines)-1]
	// WARN: Trim any excess null bytes (\x00)
	trimmedPayload := strings.TrimRight(lastLine, "\x00")
	fmt.Println("Extracted and trimmed payload data:", trimmedPayload)
	return trimmedPayload, nil
}

func checkFileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func readFile(filepath string) ([]byte, error) {
	dat, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	fmt.Print(string(dat))

	return dat, nil
}

func createFile(filepath string, data []byte) error {
	// TODO:
	// 1. [X] write file
	// 2. [X] return response (error or nil)
	err := os.WriteFile(filepath, data, 0644)
	if err != nil {
		return err
	}
	fmt.Println("Data saved!")
	return nil
}

func main() {
	var directoryPath string
	flag.StringVar(&directoryPath, "directory", "tmp", "directory where files live")

	flag.Parse()

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
		go handleRequest(conn, directoryPath)

	}

}
