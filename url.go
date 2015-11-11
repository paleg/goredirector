package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type URL struct {
	Host      string
	Domain    string
	SubDomain string
	Port      string
	Dirs      string
}

func (u URL) Base() bool {
	return u.SubDomain == "" &&
		u.Port == "" &&
		u.Dirs == ""
}

func (u URL) String() string {
	if u.SubDomain != "" {
		u.SubDomain += "."
	}
	if u.Port != "" {
		u.Port = ":" + u.Port
	}
	return fmt.Sprintf("%v%v%v%v", u.SubDomain, u.Domain, u.Port, u.Dirs)
}

func (u1 URL) EqualTo(u2 URL) bool {
	return u1.Domain == u2.Domain &&
		u1.SubDomain == u2.SubDomain &&
		u1.Port == u2.Port &&
		u1.Dirs == u2.Dirs
}

func ParseUrl(rawurl string) (URL, error) {
	var u URL
	// url.Parse can't parse urls without prefixes, lets fix that
	var fixedurl string
	if strings.Index(rawurl, "://") == -1 {
		fixedurl = "http://" + rawurl
	} else {
		fixedurl = rawurl
	}

	if parsed_url, err := url.Parse(fixedurl); err != nil {
		return u, err
	} else {
		// net/url/URL.Host is a 'host or host:port', lets split them
		if indx := strings.Index(parsed_url.Host, ":"); indx != -1 {
			u.Host = parsed_url.Host[:indx]
			u.Port = parsed_url.Host[indx+1:]
		} else {
			u.Host = parsed_url.Host
		}
		for _, c := range u.Host {
			if !((c > 44 && c < 47) || (c > 47 && c < 58) || (c > 64 && c < 91) || c == 95 || (c > 96 && c < 123)) {
				return u, errors.New(fmt.Sprintf("skipping %v because of bad character in domain name", u.Host))
			}
		}
		// https://golang.org/pkg/net/url/#URL
		// net/url:URL.Path is a decoded query path without args
		u.Dirs = parsed_url.Path
		// lets separate two-level domain from N-level subdomains (N>2)
		splitted := strings.SplitN(strings.ToLower(u.Host), ".", -1)
		if len(splitted) < 3 {
			u.Domain = u.Host
		} else {
			u.Domain = strings.Join(splitted[len(splitted)-2:len(splitted)], ".")
			u.SubDomain = strings.Join(splitted[:len(splitted)-2], ".")
		}
	}
	return u, nil
}
