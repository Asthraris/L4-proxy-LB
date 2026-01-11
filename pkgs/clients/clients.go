package clients

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"time"
)

func fireAndForget(ctx context.Context, id int, url string) {

	//dial to server
	conn, err := net.Dial("tcp", url)
	if err != nil {
		log.Println("ERR: Problem Connecting to Url")
		return
	}
	defer conn.Close()

	//send data once then exit the func so this goroutine end if that ends the connection gets closed automaticlly
	payload := fmt.Sprintf("PING %d\n", id)
	conn.Write([]byte(payload))
}

func realClient(ctx context.Context, id int, url string) {
	conn, err := net.Dial("tcp", url)
	if err != nil {
		log.Println("ERR: ", err)
		return
	}
	defer conn.Close()

	//sends 10 times
	n := rand.IntN(10)
	for i := 0; i < n; i++ {
		conn.Write([]byte("REQ\n"))
		buf := make([]byte, 1024)
		conn.Read(buf)
		time.Sleep(time.Duration(rand.IntN(1000)) * time.Millisecond)
	}
}
func streamClient(ctx context.Context, id int, url string) {

}

// for public Func of package we should have Capital first letter
func LaunchClients(ctx context.Context, numClients int) {
	//get the key first
	URL_KEY := ctx.Value(URL_KEY) //since URL_KEY is from same package no need to import it

	url, ok := URL_KEY.(string)
	if ok == false {
		log.Println("ERR: URL INJECTION PROBLEM ")
		return
	}
	//get value of key
	//for now i am just adding REGULAR clients , add variations later , with random ranges
	for n_i := range numClients {
		//creating one goroutine per client

		// Zombie client (misbehaving)
		// Long-lived streaming client
		// Idle persistent client (most realistic)
		go realClient(ctx, n_i, url)
	}
}
