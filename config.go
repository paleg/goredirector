package main

import (
	"./libredirector"
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	verbose    bool
	debug      bool
	error_log  string
	change_log string
	work_ip    []string
	allow_ip   []string
	work_id    []string
	allow_id   []string
	allow_urls string
	Categories map[string]*libredirector.Category
}

func FilterComments(in []string) (res []string) {
	for _, s := range in {
		if s != "" {
			if strings.HasPrefix(s, "#") {
				break
			}
			res = append(res, s)
		}
	}
	return
}

func (c *Config) SetOpt(category string, opt string, value string) {
	if category == "" {
		switch opt {
		case "error_log":
			c.error_log = value
		case "change_log":
			c.change_log = value
		case "work_ip":
			c.work_ip = append(c.work_ip, value)
		case "allow_ip":
			c.allow_ip = append(c.allow_ip, value)
		case "work_id":
			c.work_id = append(c.work_id, value)
		case "allow_id":
			c.allow_id = append(c.work_id, value)
		case "allow_urls":
			c.allow_urls = value
		}
	} else {
		switch opt {
		case "ban_dir":
			c.Categories[category].UrlsFile = value + "/urls"
			c.Categories[category].PcreFile = value + "/pcre"
		case "url":
			c.Categories[category].RedirUrl = value
		case "work_ip":
			c.Categories[category].WorkIP = append(c.Categories[category].WorkIP, value)
		case "allow_ip":
			c.Categories[category].AllowIP = append(c.Categories[category].AllowIP, value)
		case "work_id":
			c.Categories[category].WorkID = append(c.Categories[category].WorkID, value)
		case "allow_id":
			c.Categories[category].AllowID = append(c.Categories[category].AllowID, value)
		case "log":
			if value == "off" {
				c.Categories[category].Log = false
			}
		case "reverse":
			c.Categories[category].Reverse = true
		}
	}
}

func ReadConfig(conf string) (config Config, err error) {
	file, err := os.Open(conf)
	if err != nil {
		fmt.Printf("Failed to open config from %+v: %+v\n", conf, err)
		return
	}
	defer file.Close()

	//config := Config{}
	config.Categories = make(map[string]*libredirector.Category)

	var category string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		splitted_dash := FilterComments(strings.Split(line, " "))
		if splitted_dash != nil {
			if strings.HasPrefix(splitted_dash[0], "<") {
				category = strings.Trim(splitted_dash[0], "<>")
				config.Categories[category] = &libredirector.Category{Title: category, Log: true, Reverse: false}
			} else {
				if len(splitted_dash) == 1 {
					splitted_dash = append(splitted_dash, "")
				}
				config.SetOpt(category, splitted_dash[0], splitted_dash[1])
			}
		}
	}
	return
}
