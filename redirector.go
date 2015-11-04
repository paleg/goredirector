package main

import (
	"./libredirector"
	"bufio"
	"flag"
	"io"
	"io/ioutil"
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

var config Config

var channels map[string]chan *libredirector.Input

var ConsoleLogger *log.Logger
var ChangeLogger *log.Logger
var ChangeLoggerFile *os.File
var ErrorLogger *log.Logger
var ErrorLoggerFile *os.File

func init() {
	flag.BoolVar(&verboseFlag, "v", false, "verbose output")
	flag.BoolVar(&debugFlag, "d", true, "debug output")
	flag.StringVar(&configFlag, "c", "", "config file location")
}

func handleSignals() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP)
	for {
		sig := <-sigc
		ErrorLogger.Println("Signal received:", sig)
		switch sig {
		case syscall.SIGHUP:
			libredirector.WGConfig.Wait()
			ErrorLogger.Println("Reloading configuration")
			if newcfg, err := NewConfig(configFlag); err != nil {
				ErrorLogger.Printf("Failed to reload config: %v\n", err)
			} else {
				if err := setLogging(&newcfg); err != nil {
					// this will go to squid cache.log
					ConsoleLogger.Println("redirector| Failed to set log - '%v'", err)
				} else {
					newcfg.LoadCategories()
					config = newcfg
				}
			}
		}
	}
}

func setLoggingFile(path string, file *os.File) (logger *log.Logger, err error) {
	file.Close()
	if path != "" {
		if file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666); err != nil {
			return nil, err
		} else {
			logger = log.New(file, "", log.Ldate|log.Ltime)
		}
	} else {
		logger = log.New(ioutil.Discard, "", 0)
	}
	return logger, nil
}

func setLogging(cfg *Config) (err error) {
	if ChangeLogger, err = setLoggingFile(cfg.change_log, ChangeLoggerFile); err != nil {
		return
	}
	if ErrorLogger, err = setLoggingFile(cfg.error_log, ErrorLoggerFile); err != nil {
		return
	}
	return
}

func main() {
	ConsoleLogger = log.New(os.Stderr, "", 0)

	flag.Parse()
	if newcfg, err := NewConfig(configFlag); err != nil {
		ConsoleLogger.Fatalf("Failed to read config - '%v'", err)
	} else {
		if err := setLogging(&newcfg); err != nil {
			ConsoleLogger.Fatalf("Failed to set log - '%v'", err)
		}
		defer ChangeLoggerFile.Close()
		defer ErrorLoggerFile.Close()
		config = newcfg
	}

	config.LoadCategories()

	go handleSignals()

	// read from stdin
	reader := bufio.NewReader(os.Stdin)

	// sync write to stdout
	writer_chan := make(chan string)
	go libredirector.OutWriter(writer_chan)
	libredirector.WGMain.Add(1)

	ErrorLogger.Println("Started, ready to serve requests")

	channels = make(map[string]chan *libredirector.Input)
	for {
		if line, err := reader.ReadString('\n'); err != nil {
			if err != io.EOF {
				ConsoleLogger.Println("Failed to read string:", err)
			}
			break
		} else {
			// TODO: strings.Trim("\n\r")
			if input, err := libredirector.ParseInput(line[:len(line)-1]); err != nil {
				ConsoleLogger.Println("Failed to parse input:", err)
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
	ErrorLogger.Println("Finished")
}
