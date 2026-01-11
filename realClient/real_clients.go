package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os" //for getting additional args while running exe
	"time"
)

func takeinputs(conn net.Conn, timeout time.Duration) {
	stdin := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := stdin.ReadString('\n') //read till enter{\n} is pressed
		if err != nil {
			log.Println("peer too slow, closing")
			return
		}
		//write that line into connection
		conn.SetWriteDeadline(time.Now().Add(timeout))
		_, err = conn.Write([]byte(line))
		if err != nil {
			log.Println("write error:", err)
			return
		}
	}
}

func printRes(conn net.Conn, timeout time.Duration) {
	buff_win := make([]byte, 1024)
	for {
		conn.SetReadDeadline(time.Now().Add(timeout))
		dat, err := conn.Read(buff_win)
		if err != nil {
			if err == io.EOF {
				log.Println("server closed connection")
			} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Println("peer too slow, closing")
			} else {
				log.Println("ERR : cant read from server successfully! ")
			}
			return
		}
		fmt.Printf("RES: %s", buff_win[:dat])
	}
}

func main() {
	//Taking args as port numeber for clients to run
	if len(os.Args) != 2 {
		log.Fatalf("ERR : Server port not provided")
	}

	conn, err := net.Dial("tcp", "localhost:"+os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Printf("INFO : Connected to server :%s", os.Args[1])

	timeout := 30 * time.Second

	go takeinputs(conn, timeout)
	go printRes(conn, timeout)

	select {} //this puts main routine on sleep so it doesnt end instantly
}
