package main

import (
	"container/list"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	HeartBeatSeconds   = 5
	QueueExpireSeconds = 300
)

// concurrency safe list of connections
var conns connList

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

// connList saves connection to value of list and it's multi thread safe
type connList struct {
	*list.List
	sync.RWMutex
}

func newConnList() connList {
	return connList{
		List: list.New(),
	}
}

// connList.pushFrontWithFront pushes value to front of list and return it
func (cl *connList) PushFront(c *connection) *list.Element {
	cl.Lock()
	defer cl.Unlock()

	return cl.List.PushFront(c)
}

// connList.remove removes the element of list
func (cl *connList) Remove(e *list.Element) {
	cl.Lock()
	defer cl.Unlock()

	cl.List.Remove(e)
}

// connList.pushMessage pushes messages from client to other clients
func (cl *connList) PushMessage(conn *connection, message string) {
	cl.RLock()
	defer cl.RUnlock()

	for e := cl.List.Front(); e != nil; e = e.Next() {
		c := e.Value.(*connection)
		if c != conn {
			_, err := c.Write([]byte(message))
			if err != nil {
				fmt.Printf("Send message to %s failed(%s) and closed!", c.RemoteAddr().String(), err.Error())
				continue
			}
		}
	}
}

// queueList saves buffer queue to value of list and it's multi thread safe
type queueList struct {
	*list.List
	sync.RWMutex
}

func newQueueList() queueList {
	return queueList{
		List: list.New(),
	}
}

type queue struct {
	time   time.Time
	buffer []byte
}

// queueList.Front gets the first value of list
func (ql *queueList) Front() *list.Element {
	ql.Lock()
	defer ql.Unlock()

	return ql.List.Front()
}

// queueList.removes removes the element of list
func (ql *queueList) Remove(e *list.Element) {
	ql.Lock()
	defer ql.Unlock()

	ql.List.Remove(e)
}

// queueList.PushFront pushs value to front of list
func (ql *queueList) PushFront(q queue) {
	ql.Lock()
	defer ql.Unlock()

	ql.List.PushFront(q)
}

// queueList.DynamicQueue removes idle over a certain period of time buffer queue
func (ql *queueList) DynamicQueue() {
	ql.Lock()
	defer ql.Unlock()

	for e := ql.List.Front(); e != nil; {
		n := e.Next()
		fmt.Println(time.Now().Sub(e.Value.(queue).time).Seconds())
		if time.Now().Sub(e.Value.(queue).time) > QueueExpireSeconds*time.Second {
			ql.List.Remove(e)
			e.Value = nil
		}
		e = n
	}

	time.AfterFunc(QueueExpireSeconds*time.Second, ql.DynamicQueue)
}

func makeBuffer() []byte {
	return make([]byte, 256)
}

// makeRecycler dynamic provision and recovery of buffer
func makeRecycler() (in, out chan []byte) {
	in = make(chan []byte)
	out = make(chan []byte)
	q := newQueueList()
	go func() {
		for {

			// make new buffer when buffer list empty
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

	go q.DynamicQueue()

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

	// init conns
	conns = newConnList()

	in, out := makeRecycler()

	// keep accepting connection and handle it
	for {
		c, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		conn := newConn(c)

		e := conns.PushFront(conn)
		fmt.Printf("Accept a connect %s! \r\n", conn.RemoteAddr().String())

		go handleConnection(conn, in, out)
		go func() {
			// remove closed connection
			defer conns.Remove(e)
			heartBeat(conn)
		}()
	}
}

// handleConnection receives message and send to other clients
func handleConnection(conn *connection, in, out chan []byte) {
	buf := <-in
	for {
		i, err := conn.Read(buf)
		if err != nil {
			// connection has been disconnected and waiting to remove
			if err == io.EOF && strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Printf("%s has been disconnected! \r\n", conn.RemoteAddr().String())
				out <- buf
				return
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

			conns.PushMessage(conn, message)
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

			return
		}
	}
}
