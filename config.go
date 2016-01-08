package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type RawChange struct {
	Old string
	New string
}

type Config struct {
	error_log    string
	change_log   string
	cache_dir    string
	ADServer     []string
	ADUser       string
	ADPassword   string
	ADSearchBase string
	ADReload     int
	ADTicker     *time.Ticker
	ADTickerQuit chan struct{}
	LogHost      bool
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
	RawChanges   []RawChange
	RawChangeLog bool
	Security     *Security
}

func FilterComments(in []string) (res []string) {
	for _, s := range in {
		if s != "" {
			if strings.HasPrefix(s, "#") {
				break
			}
			res = append(res, strings.Trim(s, " \t"))
		}
	}
	return
}

func (c *Config) SetOpt(category string, values []string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if category == "" {
				err = errors.New(fmt.Sprintf("Wrong config option: %v", values))
			} else {
				err = errors.New(fmt.Sprintf("Wrong config option in '%v' category: %v", category, values))
			}
		}
	}()

	if category == "" {
		switch values[0] {
		case "error_log":
			c.error_log = values[1]
		case "change_log":
			c.change_log = values[1]
		case "cache_dir":
			c.cache_dir = values[1]
		case "ad_server":
			c.ADServer = append(c.ADServer, values[1])
		case "ad_user":
			c.ADUser = values[1]
		case "ad_password":
			c.ADPassword = values[1]
		case "ad_searchbase":
			c.ADSearchBase = values[1]
		case "ad_reload":
			c.ADReload, err = strconv.Atoi(values[1])
			if err != nil {
				return
			}
		case "work_ip":
			c.work_ip = append(c.work_ip, values[1])
		case "allow_ip":
			c.allow_ip = append(c.allow_ip, values[1])
		case "work_id":
			c.work_id = append(c.work_id, values[1])
		case "allow_id":
			c.allow_id = append(c.allow_id, values[1])
		case "allow_urls":
			c.allow_urls = values[1]
		case "write_hostname_to_log":
			c.LogHost = true
		case "raw_change":
			c.RawChanges = append(c.RawChanges, RawChange{Old: values[1], New: values[2]})
		case "raw_log":
			if values[1] == "off" {
				c.RawChangeLog = false
			}
		}
	} else if category == "SECURITY" {
		switch values[0] {
		case "url":
			c.Security.RedirUrl = values[1]
		case "enforce-https-with-hostname":
			if values[1] == "off" {
				c.Security.EnforceHTTPSHostnames = false
			}
		case "enforce-https-official-certificate":
			if values[1] == "off" {
				c.Security.EnforceHTTPSVerifiedCerts = false
			}
		case "allow-unknown-protocol-over-https":
			if values[1] == "off" {
				c.Security.AllowUnknownProtocol = false
			}
		case "check-proxy-tunnels":
			if values[1] == "off" {
				c.Security.CheckProxyTunnels = false
			}
		case "policy":
			if values[1] == "queue-checks" {
				c.Security.Policy = CheckSecuriry_Queue
			} else if values[1] == "aggressive" {
				c.Security.Policy = CheckSecuriry_Aggressive
			} else if values[1] == "log-only" {
				c.Security.Policy = CheckSecuriry_LogOnly
			} else if values[1] == "off" {
				c.Security.Policy = CheckSecuriry_Off
			}
		}
	} else {
		switch values[0] {
		case "ban_dir":
			c.Categories[category].UrlsFiles = append(c.Categories[category].UrlsFiles, values[1]+"/urls")
			c.Categories[category].PcreFiles = append(c.Categories[category].PcreFiles, values[1]+"/pcre")
		case "urls_file":
			c.Categories[category].UrlsFiles = append(c.Categories[category].UrlsFiles, values[1])
		case "pcre_file":
			c.Categories[category].PcreFiles = append(c.Categories[category].PcreFiles, values[1])
		case "url":
			c.Categories[category].RedirUrl = values[1]
		case "work_ip":
			c.Categories[category].work_ip = append(c.Categories[category].work_ip, values[1])
		case "allow_ip":
			c.Categories[category].allow_ip = append(c.Categories[category].allow_ip, values[1])
		case "work_id":
			c.Categories[category].work_id = append(c.Categories[category].work_id, values[1])
		case "allow_id":
			c.Categories[category].allow_id = append(c.Categories[category].allow_id, values[1])
		case "log":
			if values[1] == "off" {
				c.Categories[category].Log = false
			}
		case "reverse":
			c.Categories[category].Reverse = true
		case "action":
			if values[1] == "pass" {
				c.Categories[category].Action = ActionPass
			}
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

func (c *Config) ExtendIPs(ips []string) (result []*net.IPNet) {
	for _, ip := range ips {
		splitted_ip := strings.Split(ip, "/")
		if len(splitted_ip) == 1 {
			ip += "/32"
		} else if len(splitted_ip) != 2 || len(splitted_ip[1]) != 2 {
			ErrorLogger.Printf("Invalid CIDR address: %v\n", ip)
			continue
		}
		if _, ipnet, err := net.ParseCIDR(ip); err != nil {
			ErrorLogger.Printf("%v\n", err)
		} else {
			result = append(result, ipnet)
		}
	}
	return
}

func (c *Config) ExtendFromFile(list []string) (result []string) {
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
			if !c.UseAD() {
				ErrorLogger.Println("Found 'ad' prefix in config but AD support was not configured")
			}
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
		c.AllowURLs.UrlsFiles = append(c.AllowURLs.UrlsFiles, c.allow_urls)
		c.AllowURLs.Load()
	}

	c.WorkID = c.ExtendFromFile(c.work_id)
	c.WorkIP = c.ExtendIPs(c.ExtendFromFile(c.work_ip))
	c.AllowID = c.ExtendFromFile(c.allow_id)
	c.AllowIP = c.ExtendIPs(c.ExtendFromFile(c.allow_ip))

	for _, cat := range c.Categories {
		cat.WorkID = c.ExtendFromFile(cat.work_id)
		cat.WorkIP = c.ExtendIPs(c.ExtendFromFile(cat.work_ip))
		cat.AllowID = c.ExtendFromFile(cat.allow_id)
		cat.AllowIP = c.ExtendIPs(c.ExtendFromFile(cat.allow_ip))
	}
}

func (c *Config) RawChange(inurl string) (string, error) {
	for _, change := range c.RawChanges {
		if strings.Index(inurl, change.Old) != -1 {
			return strings.Replace(inurl, change.Old, change.New, 1), nil
		}
	}
	return "", errors.New("")
}
func NewConfig(conf string) (newcfg *Config, err error) {
	file, err := os.Open(conf)
	if err != nil {
		return
	}
	defer file.Close()

	newcfg = &Config{LogHost: false, RawChangeLog: true, cache_dir: "/tmp"}
	newcfg.Categories = make(map[string]*Category)
	newcfg.AllowURLs = &Category{Title: "ALLOWED_URLS", cache_dir: &newcfg.cache_dir}
	newcfg.Security = &Security{Title: "SECURITY",
		EnforceHTTPSHostnames:     true,
		EnforceHTTPSVerifiedCerts: true,
		CheckProxyTunnels:         true,
		Policy:                    CheckSecuriry_LogOnly,
		Results:                   SecurityResults{r: make(map[string]int)},
	}

	var category string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		splitted_dash := FilterComments(strings.Split(line, " "))
		if splitted_dash != nil {
			if strings.HasPrefix(splitted_dash[0], "<") {
				category = strings.Trim(splitted_dash[0], "<>")
				if category != "SECURITY" {
					newcfg.Categories[category] = &Category{Title: category, Log: true, Reverse: false, Action: ActionRedir, cache_dir: &newcfg.cache_dir}
				}
			} else {
				if err = newcfg.SetOpt(category, splitted_dash); err != nil {
					return
				}
			}
		}
	}
	return
}
