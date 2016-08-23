package main

import (
	"container/list"
	"flag"
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

type queue struct {
	time   time.Time
	buffer []byte
}

func makeBuffer() []byte {
	return make([]byte, 256)
}

// makeRecycler dynamic provision and recovery of buffer
func makeRecycler() (in, out chan []byte) {
	in = make(chan []byte)
	out = make(chan []byte)
	go func() {
		q := list.New()
		for {
			// when buffer list empty make new buffer
			if q.Len() == 0 {
				q.PushFront(queue{time: time.Now(), buffer: makeBuffer()})
			}

			e := q.Front()
			select {
			case b := <-out:
				q.PushFront(queue{time: time.Now(), buffer: b})
			case in <- e.Value.(queue).buffer:
				q.Remove(e)
			}
		}
	}()

	return
}

func main() {

	flag.Parse()
	args := flag.Args()
	if len(args) > 1 {
		fmt.Println("Too many parameters!")
		return
	}

	// default value
	port := "8070"

	// use first parameter to port
	if len(args) == 1 {
		port = args[0]
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println(err)
		fmt.Printf("Start server failed(%s)! \r\n", err.Error())
		return
	}
	fmt.Println("Start server success!")

	in, out := makeRecycler()

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

		go handleConnection(conn, in, out)
		go heartBeat(conn)
	}
}

// handleConnection receives message and send to other clients
func handleConnection(conn *connection, in, out chan []byte) {
	for {
		buf := <-in
		i, err := conn.Read(buf)
		if err != nil {
			// connection has been closed and waiting to remove
			if err == io.EOF {
				fmt.Printf("%s has been disconnected! \r\n", conn.RemoteAddr().String())
				out <- buf
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
			message = conn.RemoteAddr().String() + " : " + message
			fmt.Println(message)

			for _, v := range conns {
				if v != *conn {
					_, err := v.Write([]byte(message))
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
