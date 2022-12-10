package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"runtime"
	"sync"

	"uk.ac.bris.cs/gameoflife/gol"
)

var PORTS = []string{
	"10001", "10002", "10003", "10004", "10005", "10006", "10007", "10008",
}

// main is the function called when starting Game of Life with 'go run .'
func main() {
	runtime.LockOSThread()
	var params gol.Params

	flag.IntVar(
		&params.Threads,
		"t",
		8,
		"Specify the number of worker threads to use. Defaults to 8.")

	flag.IntVar(
		&params.ImageWidth,
		"w",
		512,
		"Specify the width of the image. Defaults to 512.")

	flag.IntVar(
		&params.ImageHeight,
		"h",
		512,
		"Specify the height of the image. Defaults to 512.")

	flag.IntVar(
		&params.Turns,
		"turns",
		10000000000,
		"Specify the number of turns to process. Defaults to 10000000000.")

	//noVis := flag.Bool(
	//	"noVis",
	//	false,
	//	"Disables the SDL window, so there is no visualisation during the tests.")

	flag.Parse()

	fmt.Println("Threads:", params.Threads)
	fmt.Println("Width:", params.ImageWidth)
	fmt.Println("Height:", params.ImageHeight)

	wg := sync.WaitGroup{}
	rpc.Register(new(gol.ServerStruct))
	rpc.HandleHTTP()
	for i := range PORTS {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			listen, err := net.Listen("tcp", "0.0.0.0:"+PORTS[index])
			if err != nil {
				fmt.Printf("port=%v start server error\n", PORTS[index])
				return
			}

			fmt.Printf("port=%v server start!\n", PORTS[index])
			http.Serve(listen, nil)
		}(i)

	}

	wg.Wait()

	//keyPresses := make(chan rune, 10)
	//events := make(chan gol.Event, 1000)
	//
	//go gol.Run(params, events, keyPresses)
	//if !(*noVis) {
	//	sdl.Run(params, events, keyPresses)
	//} else {
	//	complete := false
	//	for !complete {
	//		event := <-events
	//		switch event.(type) {
	//		case gol.FinalTurnComplete:
	//			complete = true
	//		}
	//	}
	//}
}
