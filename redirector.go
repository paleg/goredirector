package main

import (
	"./libredirector"
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
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

var config Config

func configure() error {
	if newconfig, err := ReadConfig(configFlag); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return err
	} else {
		fmt.Printf("Loaded config %#v\n", newconfig)
		for _, c := range newconfig.Categories {
			go func(c *libredirector.Category) {
				c.Load()
			}(c)
			libredirector.WGConfig.Add(1)
		}
		config = newconfig
		return nil
	}
}

func handleSignals() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP)
	for {
		sig := <-sigc
		fmt.Println("Signal received:", sig)
		switch sig {
		case syscall.SIGHUP:
			libredirector.WGConfig.Wait()
			fmt.Println("Reloading configuration")
			configure()
		}
	}
}

func main() {
	flag.Parse()
	if err := configure(); err != nil {
		os.Exit(1)
	}

	logger := log.New(os.Stderr, "goredirector|", 0)

	go handleSignals()

	// read from stdin
	reader := bufio.NewReader(os.Stdin)

	// sync write to stdout
	writer_chan := make(chan string)
	go libredirector.OutWriter(writer_chan)
	libredirector.WGMain.Add(1)

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
					libredirector.WGMain.Add(1)
				}
				channels[input.Chanid] <- &input
			}
		}
	}

	close(writer_chan)
	for _, ch := range channels {
		close(ch)
	}
	libredirector.WGConfig.Wait()
	libredirector.WGMain.Wait()
}
