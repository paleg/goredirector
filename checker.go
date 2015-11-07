package main

import (
	"fmt"
	"net"
	"sort"
)

func SearchID(ids []string, id string) bool {
	i := sort.SearchStrings(ids, id)
	return i < len(ids) && ids[i] == id
}

func SearchIP(ips []*net.IPNet, ip net.IP) bool {
	for _, ipnet := range ips {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

func Pass(out chan string, reason string) {
	ConsoleLogger.Printf("Passing request: %v\n", reason)
	out <- "ERR"
}

func Checker(id string, in chan *Input, out chan string) {
	ErrorLogger.Printf("Checker%v started\n", id)
	defer WGMain.Done()
	for input := range in {
		if url, err := ParseUrl(input.RawUrl); err != nil {
			ErrorLogger.Printf("ParseUrl error: %+v\n", err)
			out <- input.Raw
		} else {
			if len(config.WorkID) > 0 && !SearchID(config.WorkID, input.User) {
				Pass(out, "global work id is not null and user ident is not in")
				continue
			}
			if len(config.AllowID) > 0 && SearchID(config.AllowID, input.User) {
				Pass(out, "global allow id is not null and user ident is in")
				continue
			}
			if len(config.WorkIP) > 0 && !SearchIP(config.WorkIP, input.IP) {
				Pass(out, "global work ip is not null and user ip is not in")
				continue
			}
			if len(config.AllowIP) > 0 && SearchIP(config.AllowIP, input.IP) {
				Pass(out, "global allow ip is not null and user ip is in")
				continue
			}
			for _, cat := range config.Categories {
				ConsoleLogger.Printf("Checking %v\n", cat.Title)
				if len(cat.WorkID) > 0 && !SearchID(cat.WorkID, input.User) {
					ConsoleLogger.Printf("'%v' work id is not null and user ident is not in", cat.Title)
					continue
				}
				if len(cat.AllowID) > 0 && SearchID(cat.AllowID, input.User) {
					ConsoleLogger.Printf("'%v' allow id is not null and user ident is in", cat.Title)
					continue
				}
				if len(cat.WorkIP) > 0 && !SearchIP(cat.WorkIP, input.IP) {
					ConsoleLogger.Printf("'%v' work ip is not null and user ip is not in", cat.Title)
					continue
				}
				if len(cat.AllowIP) > 0 && SearchIP(cat.AllowIP, input.IP) {
					ConsoleLogger.Printf("'%v' allow ip is not null and user ip is in", cat.Title)
					continue
				}
			}
			out <- fmt.Sprintf("%#v", url)
		}
	}
	ErrorLogger.Printf("Checker%v closed\n", id)
}
