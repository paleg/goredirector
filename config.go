package main

import (
	"bufio"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	error_log    string
	change_log   string
	ADServer     []string
	ADUser       string
	ADPassword   string
	ADSearchBase string
	ADReload     int
	ADTicker     *time.Ticker
	ADTickerQuit chan struct{}
	WorkIP       []*net.IPNet
	work_ip      []string
	AllowIP      []*net.IPNet
	allow_ip     []string
	WorkID       []string
	work_id      []string
	AllowID      []string
	allow_id     []string
	allow_urls   string
	AllowURLs    *Category
	Categories   map[string]*Category
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

func (c *Config) SetOpt(category string, opt string, value string) (err error) {
	if category == "" {
		switch opt {
		case "error_log":
			c.error_log = value
		case "change_log":
			c.change_log = value
		case "ad_server":
			c.ADServer = append(c.ADServer, value)
		case "ad_user":
			c.ADUser = value
		case "ad_password":
			c.ADPassword = value
		case "ad_searchbase":
			c.ADSearchBase = value
		case "ad_reload":
			c.ADReload, err = strconv.Atoi(value)
			if err != nil {
				return
			}
		case "work_ip":
			c.work_ip = append(c.work_ip, value)
		case "allow_ip":
			c.allow_ip = append(c.allow_ip, value)
		case "work_id":
			c.work_id = append(c.work_id, value)
		case "allow_id":
			c.allow_id = append(c.allow_id, value)
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
			c.Categories[category].work_ip = append(c.Categories[category].work_ip, value)
		case "allow_ip":
			c.Categories[category].allow_ip = append(c.Categories[category].allow_ip, value)
		case "work_id":
			c.Categories[category].work_id = append(c.Categories[category].work_id, value)
		case "allow_id":
			c.Categories[category].allow_id = append(c.Categories[category].allow_id, value)
		case "log":
			if value == "off" {
				c.Categories[category].Log = false
			}
		case "reverse":
			c.Categories[category].Reverse = true
		}
	}
	return
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

func ExtendIPs(ips []string) (result []*net.IPNet) {
	for _, ip := range ips {
		splitted_ip := strings.Split(ip, "/")
		if len(splitted_ip) == 1 {
			ip += "/32"
		} else if len(splitted_ip) != 2 || len(splitted_ip[1]) != 2 {
			ErrorLogger.Printf("Wrong ip address - %v\n", ip)
		}
		if _, ipnet, err := net.ParseCIDR(ip); err != nil {
			ErrorLogger.Printf("Wrong CIDR address - %v: \v\n", ip, err)
		} else {
			result = append(result, ipnet)
		}
	}
	return
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
		} else if strings.HasPrefix(s, "ad:") && len(s) > 3 {
			// will be added later in ad goroutine
			continue
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

	c.WorkID = ExtendFromFile(c.work_id)
	ErrorLogger.Printf("c.WorkID = %#v\n", c.WorkID)
	c.WorkIP = ExtendIPs(ExtendFromFile(c.work_ip))
	ErrorLogger.Printf("c.WorkIP = %v\n", c.WorkIP)
	c.AllowID = ExtendFromFile(c.allow_id)
	ErrorLogger.Printf("c.AllowID = %#v\n", c.AllowID)
	c.AllowIP = ExtendIPs(ExtendFromFile(c.allow_ip))
	ErrorLogger.Printf("c.AllowIP = %v\n", c.AllowIP)

	for _, cat := range c.Categories {
		cat.WorkID = ExtendFromFile(cat.work_id)
		ErrorLogger.Printf("%v WorkID = %#v\n", cat.Title, cat.WorkID)
		cat.WorkIP = ExtendIPs(ExtendFromFile(cat.work_ip))
		ErrorLogger.Printf("%v WorkIP = %v\n", cat.Title, cat.WorkIP)
		cat.AllowID = ExtendFromFile(cat.allow_id)
		ErrorLogger.Printf("%v AllowID = %#v\n", cat.Title, cat.AllowID)
		cat.AllowIP = ExtendIPs(ExtendFromFile(cat.allow_ip))
		ErrorLogger.Printf("%v AllowIP = %v\n", cat.Title, cat.AllowIP)
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
				if err = newcfg.SetOpt(category, splitted_dash[0], splitted_dash[1]); err != nil {
					return
				}
			}
		}
	}
	return
}
