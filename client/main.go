package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

const (
	HeartBeatSeconds = 5
)

type connection struct {
	net.Conn
	Signal chan struct{}
}

func newConn(c net.Conn) *connection {
	return &connection{
		Conn:   c,
		Signal: make(chan struct{}, 1),
	}
}

func main() {
	c, err := net.Dial("tcp", "127.0.0.1:8070")
	if err != nil {
		fmt.Println("Connect failed!")
		return
	}
	fmt.Println("Connect success!")

	conn := newConn(c)
	go readConnection(conn)
	go heartBeat(conn)

	message := ""

	// keep receiving message from keyboard
	for {
		fmt.Scanln(&message)
		if len([]byte(message)) > 256 {
			fmt.Println("Message too long!")
			continue
		}

		conn.Signal <- struct{}{}

		_, err := conn.Write([]byte("*" + message))
		if err != nil {
			fmt.Println("Has been disconnected from the server!")
			os.Exit(1)
		}
	}
}

// readConnection receives message from other clients.
func readConnection(conn *connection) {
	buf := make([]byte, 256)
	for {
		i, err := conn.Read(buf)
		if err == nil {
			fmt.Println(conn.RemoteAddr().String(), ":", string(buf[0:i]))
		}
	}
}

// heartBeat send message in fix time(HeartBeatSeconds * time.Second) or will be closed.
// It is used to judge whether this client is alive or not.
func heartBeat(conn *connection) {
	for {
		select {
		case <-conn.Signal:

		case <-time.After(HeartBeatSeconds * time.Second):
			_, err := conn.Write([]byte("*"))
			if err != nil {
				fmt.Println("Has been disconnected from the server!")
				os.Exit(1)
			}
		}
	}
}
