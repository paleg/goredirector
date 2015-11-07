package main

import "fmt"

func OutWriter(out chan string) {
	defer WGMain.Done()
	for data := range out {
		fmt.Printf("%s\n", data)
	}
}
