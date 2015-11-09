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

func Pass(id string, out chan string, reason string) {
	//ConsoleLogger.Printf("Passing request: %v\n", reason)
	out <- id + " ERR"
}

func RawRedirect(id string, out chan string, input *Input, redir_url string) {
	out <- id + " OK rewrite-url=" + redir_url
	if config.RawChangeLog {
		ChangeLogger.Printf("RAW_CHANGE: %s %s %s %s -> %s", input.IP, input.Host, input.User, input.RawUrl, redir_url)
	}
}

func (cat *Category) Redirect(id string, out chan string, input *Input, reason string) {
	if cat.Action == ActionPass {
		Pass(id, out, reason)
		return
	}

	r := strings.NewReplacer(
		"#URL#", input.RawUrl,
		"#IP#", input.IP.String(),
		"#IDENT#", input.User,
		"#METHOD#", input.Method,
		"#SECTION#", cat.Title,
	)
	redir_url := r.Replace(cat.RedirUrl)
	out <- id + " OK rewrite-url=" + redir_url

	if cat.Log {
		ChangeLogger.Printf("%s: %s %s %s %s (%s)", cat.Title, input.IP, input.Host, input.User, input.RawUrl, reason)
	}
}

func Checker(id string, in chan *Input, out chan string) {
	ErrorLogger.Printf("Checker%v started\n", id)
	defer WGMain.Done()
	for input := range in {
		if len(config.RawChanges) > 0 {
			if newurl, err := config.RawChange(input.RawUrl); err == nil {
				RawRedirect(id, out, input, newurl)
				continue
			}
		}

		if len(config.WorkID) > 0 && !SearchID(config.WorkID, input.User) {
			Pass(id, out, "global work id is not null and user ident is not in")
			continue
		}
		if len(config.AllowID) > 0 && SearchID(config.AllowID, input.User) {
			Pass(id, out, "global allow id is not null and user ident is in")
			continue
		}
		if len(config.WorkIP) > 0 && !SearchIP(config.WorkIP, input.IP) {
			Pass(id, out, "global work ip is not null and user ip is not in")
			continue
		}
		if len(config.AllowIP) > 0 && SearchIP(config.AllowIP, input.IP) {
			Pass(id, out, "global allow ip is not null and user ip is in")
			continue
		}

		parsed_url, err_parse := ParseUrl(input.RawUrl)
		if err_parse != nil {
			//ConsoleLogger.Printf("Can not parse input url: %v\n", err)
			Pass(id, out, "failed to parse input url")
			continue
		}

		if len(config.AllowURLs.Urls) > 0 {
			if hit, hitrule := config.AllowURLs.CheckURL(&parsed_url); hit {
				Pass(id, out, fmt.Sprintf("global allow_urls (%s)", hitrule))
				continue
			}
		}

		found := false
		for _, cat := range config.Categories {
			if len(cat.WorkID) > 0 && !SearchID(cat.WorkID, input.User) {
				//ConsoleLogger.Printf("'%v' work id is not null and user ident is not in", cat.Title)
				continue
			}
			if len(cat.AllowID) > 0 && SearchID(cat.AllowID, input.User) {
				//ConsoleLogger.Printf("'%v' allow id is not null and user ident is in", cat.Title)
				continue
			}
			if len(cat.WorkIP) > 0 && !SearchIP(cat.WorkIP, input.IP) {
				//ConsoleLogger.Printf("'%v' work ip is not null and user ip is not in", cat.Title)
				continue
			}
			if len(cat.AllowIP) > 0 && SearchIP(cat.AllowIP, input.IP) {
				//ConsoleLogger.Printf("'%v' allow ip is not null and user ip is in", cat.Title)
				continue
			}

			if len(cat.Urls) > 0 {
				if hit, hitrule := cat.CheckURL(&parsed_url); (hit && !cat.Reverse) || (!hit && cat.Reverse) {
					cat.Redirect(id, out, input, fmt.Sprintf("urls rule: %s", hitrule))
					found = true
					break
				}
			}

			if len(cat.Pcre) > 0 {
				if hit, hitid := cat.CheckPCRE(input.RawUrl); (hit && !cat.Reverse) || (!hit && cat.Reverse) {
					cat.Redirect(id, out, input, fmt.Sprintf("pcre rule: #%v", hitid))
					found = true
					break
				}
			}
		}

		if !found {
			Pass(id, out, "url not found in black lists")
		}
	}
	ErrorLogger.Printf("Checker%v closed\n", id)
}
