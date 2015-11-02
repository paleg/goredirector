package main

import (
	"./libredirector"
	"bufio"
	"io"
	"log"
	"os"
)

var channels map[string]chan *libredirector.Input

func main() {
	logger := log.New(os.Stderr, "goredirector|", 0)
	// read from stdin
	reader := bufio.NewReader(os.Stdin)

	// sync write to stdout
	writer_chan := make(chan string)
	go libredirector.OutWriter(writer_chan)
	libredirector.WG.Add(1)

	category := libredirector.Category{Title: "AV",
		UrlsFile: "/home/oleg/iCloud/projects/goredirector/banlists/av/urls",
		PcreFile: "/home/oleg/iCloud/projects/goredirector/banlists/av/pcre",
	}
	category.Load()
	channels = make(map[string]chan *libredirector.Input)
	for {
		if line, err := reader.ReadString('\n'); err != nil {
			if err != io.EOF {
				logger.Println("Failed to read string:", err)
			}
			break
		} else {
			// TODO: strings.Trim("\n\r")
			if input, err := libredirector.ParseInput(line[:len(line)-1]); err != nil {
				logger.Println("Failed to parse input:", err)
			} else {
				// dynamically create separate goroutine for each squid chan-id
				if _, ok := channels[input.Chanid]; !ok {
					channels[input.Chanid] = make(chan *libredirector.Input)
					go libredirector.Checker(input.Chanid, channels[input.Chanid], writer_chan)
					libredirector.WG.Add(1)
				}
				channels[input.Chanid] <- &input
			}
		}
	}

	close(writer_chan)
	for _, ch := range channels {
		close(ch)
	}
	libredirector.WG.Wait()
}
