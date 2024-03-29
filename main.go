package main

import (
	"fmt"
	"go-rabbit/clients/rabbit"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	runtime.GOMAXPROCS(4)

	// set up graceful shutdown
	sysSigChannel := make(chan os.Signal, 1)
	sysSigProcessedChannel := make(chan bool, 1)
	signal.Notify(sysSigChannel, syscall.SIGINT)

	go func() {
		signal := <-sysSigChannel
		fmt.Println("received signal:", signal)

		err := rabbit.Disconnect()
		if err != nil {
			fmt.Println(err)
		}
		sysSigProcessedChannel <- true
	}()

	var err error

	fmt.Println("creating queue")
	err = rabbit.CreateQueue("hello")
	if err != nil {
		panic(err)
	}

	for i := 0; i < 4; i++ {
		err = rabbit.Consume("hello", processHellos)
		if err != nil {
			panic(err)
		}
	}

	for i := 0; i < 200; i++ {
		fmt.Println("publishing message")

		err = rabbit.Publish("hello", "hi")
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("waiting for exit signal...")
	<-sysSigProcessedChannel
	fmt.Println("exiting")
}

func processHellos(message string, ack func() error, nack func(bool) error) {
	// simulate work. sleep for up to 500ms
	// sleepDurationMillis := time.Duration(rand.Intn(500)) * time.Millisecond
	// time.Sleep(sleepDurationMillis)

	rabbit.Publish("hello", "hi")

	err := ack()
	if err != nil {
		fmt.Println("TFK", err)
	}
}
