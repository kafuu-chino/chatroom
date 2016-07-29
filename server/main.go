package main

import (
	"fmt"
	"net"
)

var conns []net.Conn = []net.Conn{}

func main() {
	ln, err := net.Listen("tcp", ":8070")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("start server!")
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		conns = append(conns, conn)
		fmt.Println("accept a connect!")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		i, err := conn.Read(buf)
		if err == nil {
			buf = buf[0:i]
			fmt.Println(conn.RemoteAddr().String(), ":", string(buf))
		}
		for _, v := range conns {
			if v != conn {
				v.Write(buf)
			}
		}
	}
}
