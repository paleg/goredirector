package main

import (
	"./libredirector"
	"bufio"
	"flag"
	"fmt"
	"github.com/spf13/viper"
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
	flag.StringVar(&configFlag, "c", "", "config file location")
}

func main() {
	flag.Parse()

	viper.SetDefault("gomaxprocs", 0)
	viper.SetDefault("verbose", false)
	viper.SetDefault("debug", false)

	if configFlag != "" {
		viper.SetConfigFile(configFlag)
	} else {
		viper.SetConfigName("redirector")
		viper.AddConfigPath("/etc/squid3")
		viper.AddConfigPath("/etc/squid")
		viper.AddConfigPath("./")
	}
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	fmt.Printf("using config from %v\n", viper.ConfigFileUsed())
	return

	logger := log.New(os.Stderr, "goredirector|", 0)
	// read from stdin
	reader := bufio.NewReader(os.Stdin)

	// sync write to stdout
	writer_chan := make(chan string)
	go libredirector.OutWriter(writer_chan)
	libredirector.WG.Add(1)

	category_av := libredirector.Category{Title: "AUDIO-VIDEO",
		UrlsFile: "banlists/audio-video/urls",
	}
	category_porno := libredirector.Category{Title: "PORNO",
		UrlsFile: "banlists/porn/urls",
	}
	category_chat := libredirector.Category{Title: "CHATS",
		UrlsFile: "banlists/chats/urls",
	}
	category_proxy := libredirector.Category{Title: "PROXY",
		UrlsFile: "banlists/proxy/urls",
	}
	categories := []libredirector.Category{category_av, category_porno, category_chat, category_proxy}
	for _, c := range categories {
		go func(c libredirector.Category) {
			c.Load()
		}(c)
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
