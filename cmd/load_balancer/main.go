package main

import (
	"bufio"

	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// TODO ADD TICKER
const IDLE_TIMEOUT = 30 * time.Second

type server struct {
	Address      string
	Max_capacity uint16
}

func main() {
	//why ?
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	fmt.Println("----------------------------------------------------------------------")
	fmt.Println("---------------------l4 LOAD BALANCER Simulation ---------------------")
	fmt.Println("----------------------------------------------------------------------")

	var lb_url string
	fmt.Print("Load-balancer's <IP:port> - ")
	fmt.Scan(&lb_url)

	lb_ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//ADD SERVER DETAILS like url , max limits, also write it as map with servers url
	var servers_details = []server{
		{"localhost:9090", 100},
		{"localhost:9091", 200},
		{"localhost:9092", 300},
		{"localhost:9093", 400},
	}

	//Making LB LISTEN
	load_balancer, err := net.Listen("tcp", lb_url)
	if err != nil {
		log.Fatalln("ERR:", err)
		// return;
		// fatal automaticlly returns with code -1
	}
	defer load_balancer.Close()

	fmt.Println("Started listioning")

	go acceptClients(load_balancer, servers_details)
	go uiLayer(cancel)

	//this ctx.Done() blocks code at this line untill done channel is signaled to unblock
	<-lb_ctx.Done()
}

func uiLayer(cancel context.CancelFunc) {
	reader := bufio.NewReader(os.Stdin)
	for {

		//is line se goroutine block hota hai till user enters hence i used this too as goruotine which whill work unless stoped
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println("input error:", err)
			continue
		}

		line = strings.TrimSpace(line)

		switch line {
		case "h":
			//print help s UI
		case "c":
			//various calculatiosn and profile options
		case "q":
			//this will signal every child tree to end goroutine func directly and also this ctx.Done channel
			cancel()
			return
		}
	}
}

// this func runs till end and accepts any connection requests on the server from clients and send them into individuallu goroutues to their handler func
func acceptClients(load_balancer net.Listener, servers_details []server) {
	//declared only once
	num_servers := len(servers_details)
	current_conn := make([]uint16, num_servers)
	var current_index uint8 //since i dont think we need not more than 255
	for {

		// this waits unitill OS sends that someone wants to connect signal
		conn, err := load_balancer.Accept()
		if err != nil {
			//when load_balancer is closed after main ended this gets closed
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("accept error: %v", err)
			continue
		}

		//check avaible connections for server
		// 1-p persistant checker future add n-p
		if current_conn[current_index] < servers_details[current_index].Max_capacity {
			current_conn[current_index] += 1
			go sessionHandler(conn, servers_details[current_index].Address)
		}
		current_index = (current_index + 1) % uint8(num_servers)

		//add deadlines for queue

		// Add and QUEUing system before sending to handle its ession with server
		//if any server is free select algo them create there session
		//Dont worry aman , kyuki OS level pe his agar tcp buffer fill hoga before read thennos stops taking more data and waits till its gets read untill timeout
		//check deadlines for queuninh connections

	}
}

func sessionHandler(C_conn net.Conn, server_add string) {
	defer C_conn.Close()

	// create an channel

	//add this to queue may be channel, which will inject res to main servers

	//link btw lb & server <maybe connected with this go routine  so both has same life line>

	//check deadlines if not dead
	//measure time served
	//read and write simulatenously untill contection is closed or ideal for long

	//after deadline idol
	//close both connection
}
