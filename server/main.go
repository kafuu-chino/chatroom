package main

import (
	"fmt"
	"io"
	"net"
	"time"
)

const (
	HeartBeatSeconds = 5
)

var conns []connection = []connection{}

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
	ln, err := net.Listen("tcp", ":8070")
	if err != nil {
		fmt.Println(err)
		fmt.Printf("Start server failed(%s)! \r\n", err.Error())
		return
	}
	fmt.Println("Start server success!")

	// keep accepting connection and handle it
	for {
		c, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		conn := newConn(c)

		conns = append(conns, *conn)
		fmt.Printf("Accept a connect %s! \r\n", conn.RemoteAddr().String())
		go handleConnection(conn)
		go heartBeat(conn)
	}
}

// handleConnection receives message and send to other clients
func handleConnection(conn *connection) {
	buf := make([]byte, 256)
	for {
		i, err := conn.Read(buf)
		if err != nil {
			// connection has been closed and waiting to remove
			if err == io.EOF {
				fmt.Printf("%s has been disconnected! \r\n", conn.RemoteAddr().String())
				break
			}

			// wait and try again before the connection closed
			fmt.Printf("Read data from %s failed(%s)! \r\n", conn.RemoteAddr().String(), err.Error())
			time.Sleep(HeartBeatSeconds * time.Second)
			continue
		}

		conn.Signal <- struct{}{}

		// first char just use in heart beat and it's not message
		message := string(buf[0:i])[1:]

		if message != "" {
			fmt.Println(conn.RemoteAddr().String(), ":", message)

			for _, v := range conns {
				if v != *conn {
					_, err := v.Write(buf[0:i])
					if err != nil {
						fmt.Printf("Send message to %s failed(%s) and closed!", conn.RemoteAddr().String(), err.Error())
						continue
					}
				}
			}
		}
	}
}

// heartBeat receives signal in fix time(3 * HeartBeatSeconds * time.Second) or closes the connection.
// It is used to judge whether the client is alive or not.
func heartBeat(conn *connection) {
	for {
		select {
		case <-conn.Signal:

		case <-time.After(3 * HeartBeatSeconds * time.Second):

			err := conn.Close()
			if err != nil {
				fmt.Printf("close %s failed(%s)! \r\n", conn.RemoteAddr().String(), err.Error())
			} else {
				fmt.Printf("%s has been closed! \r\n", conn.RemoteAddr().String())
			}

			// remove closed connection
			for i, v := range conns {
				if v == *conn {
					conns = append(conns[:i], conns[i+1:]...)
				}
			}

			return
		}
	}
}
