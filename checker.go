package main

import (
	"fmt"
	"net"
	"sort"
	"strings"
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

func (cat *Category) Redirect(out chan string, input *Input, reason string) {
	r := strings.NewReplacer(
		"#URL#", input.RawUrl,
		"#IP#", input.IP.String(),
		"#IDENT#", input.User,
		"#METHOD#", input.Method,
		"#SECTION#", cat.Title,
	)
	redir_url := r.Replace(cat.RedirUrl)
	ConsoleLogger.Printf("Redirecting request: %v\n", reason)
	out <- "OK rewrite-url=" + redir_url

	if cat.Log {
		ChangeLogger.Printf("%s: %s %s %s %s (%s)", cat.Title, input.IP, input.Host, input.User, input.RawUrl, reason)
	}
}

func Checker(id string, in chan *Input, out chan string) {
	ErrorLogger.Printf("Checker%v started\n", id)
	defer WGMain.Done()
	for input := range in {
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
		found := false
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

			if parsed_url, err := ParseUrl(input.RawUrl); err != nil {
				ConsoleLogger.Printf("Can not parse input url: %v\n", err)
				Pass(out, "failed to parse input url")
				found = true
				break
			} else if hit, rule := cat.CheckURL(&parsed_url); hit {
				cat.Redirect(out, input, fmt.Sprintf("urls rule: %s", rule))
				found = true
				break
			}

			if hit, id := cat.CheckPCRE(input.RawUrl); hit {
				cat.Redirect(out, input, fmt.Sprintf("pcre rule: #%v", id))
				found = true
				break
			}
		}
		if !found {
			Pass(out, "url not found in black lists")
		}
	}
	ErrorLogger.Printf("Checker%v closed\n", id)
}
