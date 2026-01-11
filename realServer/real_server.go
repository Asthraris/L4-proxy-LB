package main

import (
	"log"
	"net"
	"os"
	"time"
)

const IDLE_TIMEOUT = 30 * time.Second

func handleClients(conn net.Conn) {
	defer conn.Close() //goroutines runs till the program ends and if it ends its going to close the socket
	//this will close the connection after this handle func is done for this input
	//FUTURE : add Close to acc , to req of clients
	c_host, c_port, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		log.Println("ERR: read client address", err)
		//this will send msg to client that his address is not readable
		_, _ = conn.Write([]byte("ERR: Address Not Readable !\n"))
		//close the unkown connection and exit
		return
	}
	log.Println("Info: client connected:", c_host+":"+c_port)

	buff_win := make([]byte, 1024) //creating an window which can read from the connection media/socket
	// var buff [1024]byte ; STACK is less friendly here since its copy opperation is slower than heap

	//if data from client is smaller than 1024 bytes only one time read is needed but for more data we need to for loop which prints data till completion
	for {
		dat, err := conn.Read(buff_win)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Println("INFO : Connection ended ")
			} else {
				log.Println("ERR : cant read from clients successfully! ")
			}
			return
		}
		//print data till dat
		log.Printf("[%s:%s ] : %s", c_host, c_port, buff_win[:dat])

		//FUTURE: LATER ADD SERVER RESPONSE ACC TO PROTOCOL SLEEP
		//server doesnt close go routines and open again it just waits/process than sends in same routine to remove creation complexity
	}

}

func main() {
	//Args[0] is exe name
	//Args[1] is Server port
	//Args[2] is server limit[creating arctifical]
	if len(os.Args) != 3 {
		log.Fatalf("ERR: Less Arg provide : exe <port> <rate>")
	}
	server, err := net.Listen("tcp", "localhost:"+os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	//while True loop saves server to terminate instantly
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("ERR : Not accepting the client")
		}

		//created an concurrent goroutine
		go handleClients(conn)
	}
}
