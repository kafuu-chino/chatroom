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
		fmt.Println("Start server failed")
		return
	}
	fmt.Println("Start server success!")
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		conns = append(conns, conn)
		fmt.Printf("Accept a connect %s! \r\n", conn.RemoteAddr().String())
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	buf := make([]byte, 256)
	for {
		i, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("Read data from %s failed(%s)! \r\n", conn.RemoteAddr().String(), err.Error())
			continue
		}

		fmt.Println(conn.RemoteAddr().String(), ":", string(buf[0:i]))

		for _, v := range conns {
			if v != conn {
				v.Write(buf[0:i])
			}
		}
	}
}
