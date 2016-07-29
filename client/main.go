package main

import (
	"fmt"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8070")
	if err != nil {
		fmt.Println(err)
		fmt.Println("connect failed!")
		return
	}
	fmt.Println("connect success!")
	go readConnection(conn)

	content := ""

	for {
		fmt.Scanln(&content)
		conn.Write([]byte(content))
	}
}

func readConnection(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		i, err := conn.Read(buf)
		if err == nil {
			buf = buf[0:i]
			fmt.Println(conn.RemoteAddr().String(), ":", string(buf))
		}
	}
}
