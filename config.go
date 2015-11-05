package main

import (
	"bufio"
	"os"
	"sort"
	"strings"
)

type Config struct {
	verbose    bool
	debug      bool
	error_log  string
	change_log string
	WorkIP     []string
	AllowIP    []string
	WorkID     []string
	AllowID    []string
	allow_urls string
	AllowURLs  *Category
	Categories map[string]*Category
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
			c.WorkIP = append(c.WorkIP, value)
		case "allow_ip":
			c.AllowIP = append(c.AllowIP, value)
		case "work_id":
			c.WorkID = append(c.WorkID, value)
		case "allow_id":
			c.AllowID = append(c.AllowID, value)
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

func (c *Config) LoadCategories(sync bool) {
	for _, cat := range c.Categories {
		WGCategories.Add(1)
		go func(cat *Category) {
			cat.Load()
		}(cat)
	}
	if sync {
		WGCategories.Wait()
	}
}

func ExtendFromFile(list []string) (result []string) {
	for _, s := range list {
		if strings.HasPrefix(s, "f:") && len(s) > 2 {
			filename := s[2:]
			if file, err := os.Open(filename); err != nil {
				ErrorLogger.Printf("Failed to load '%+v': %+v\n", filename, err)
			} else {
				defer file.Close()
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					result = append(result, strings.ToLower(scanner.Text()))
				}
			}
		} else {
			result = append(result, strings.ToLower(s))
		}
	}
	sort.Strings(result)
	return
}

func (c *Config) LoadFiles() {
	if c.allow_urls != "" {
		WGCategories.Add(1)
		c.AllowURLs = new(Category)
		c.AllowURLs.Title = "ALLOWED_URLS"
		c.AllowURLs.UrlsFile = c.allow_urls
		c.AllowURLs.Load()
	}

	c.WorkID = ExtendFromFile(c.WorkID)
	ErrorLogger.Printf("c.WorkID = %#v\n", c.WorkID)
	c.WorkIP = ExtendFromFile(c.WorkIP)
	ErrorLogger.Printf("c.WorkIP = %#v\n", c.WorkIP)
	c.AllowID = ExtendFromFile(c.AllowID)
	ErrorLogger.Printf("c.AllowID = %#v\n", c.AllowID)
	c.AllowIP = ExtendFromFile(c.AllowIP)
	ErrorLogger.Printf("c.AllowIP = %#v\n", c.AllowIP)

	for _, cat := range c.Categories {
		cat.WorkID = ExtendFromFile(cat.WorkID)
		ErrorLogger.Printf("%v WorkID = %#v\n", cat.Title, cat.WorkID)
		cat.WorkIP = ExtendFromFile(cat.WorkIP)
		ErrorLogger.Printf("%v WorkIP = %#v\n", cat.Title, cat.WorkIP)
		cat.AllowID = ExtendFromFile(cat.AllowID)
		ErrorLogger.Printf("%v AllowID = %#v\n", cat.Title, cat.AllowID)
		cat.AllowIP = ExtendFromFile(cat.AllowIP)
		ErrorLogger.Printf("%v AllowIP = %#v\n", cat.Title, cat.AllowIP)
	}
}

func NewConfig(conf string) (newcfg *Config, err error) {
	file, err := os.Open(conf)
	if err != nil {
		return
	}
	defer file.Close()

	newcfg = new(Config)
	newcfg.Categories = make(map[string]*Category)

	var category string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		splitted_dash := FilterComments(strings.Split(line, " "))
		if splitted_dash != nil {
			if strings.HasPrefix(splitted_dash[0], "<") {
				category = strings.Trim(splitted_dash[0], "<>")
				newcfg.Categories[category] = &Category{Title: category, Log: true, Reverse: false}
			} else {
				if len(splitted_dash) == 1 {
					splitted_dash = append(splitted_dash, "")
				}
				newcfg.SetOpt(category, splitted_dash[0], splitted_dash[1])
			}
		}
	}
	return
}
