package main

import (
	"./libredirector"
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

var (
	verboseFlag bool
	debugFlag   bool
	configFlag  string
)

var channels map[string]chan *libredirector.Input

func init() {
	flag.BoolVar(&verboseFlag, "v", false, "verbose output")
	flag.BoolVar(&debugFlag, "d", true, "debug output")
	flag.StringVar(&configFlag, "c", "", "config file location")
}

func main() {
	flag.Parse()
	ReadConfig(configFlag)
	fmt.Printf("Loaded config %#v\n", config.Categories["SOCNET"])

	logger := log.New(os.Stderr, "goredirector|", 0)
	// read from stdin
	reader := bufio.NewReader(os.Stdin)

	// sync write to stdout
	writer_chan := make(chan string)
	go libredirector.OutWriter(writer_chan)
	libredirector.WG.Add(1)

	for _, c := range config.Categories {
		go func(c *libredirector.Category) {
			c.Load()
		}(c)
		libredirector.WG.Add(1)
	}

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
