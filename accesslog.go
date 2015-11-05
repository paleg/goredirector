package main

import (
	"bufio"
	"fmt"
	"os"
)

var logc chan string

func AccessLog(s string) {
	logc <- s
}

func InitAccessLog(path string, out chan string) (err error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file:", err)
	}
	accesswriter := bufio.NewWriter(file)

	logc = make(chan string)

	go func() {
		for data := range logc {
			accesswriter.WriteString(data)
		}
		defer WGMain.Done()
		defer file.Close()
	}()
	WGMain.Add(1)
	return
}
