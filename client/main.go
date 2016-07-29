package main

import (
	"fmt"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8070")
	if err != nil {
		fmt.Println(err)
		fmt.Println("Connect failed!")
		return
	}
	fmt.Println("Connect success!")
	go readConnection(conn)

	content := ""

	for {
		fmt.Scanln(&content)
		if len([]byte(content)) > 256 {
			fmt.Println("Content too long!")
			continue
		}
		conn.Write([]byte(content))
	}
}

func readConnection(conn net.Conn) {
	buf := make([]byte, 256)
	for {
		i, err := conn.Read(buf)
		if err == nil {
			fmt.Println(conn.RemoteAddr().String(), ":", string(buf[0:i]))
		}
	}
}
