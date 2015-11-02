package libredirector

import (
	//"errors"
	//"fmt"
	"net/url"
	"strings"
)

type URL struct {
	RawUrl    string
	Host      string
	Domain    string
	SubDomain string
	Port      string
	Path      string
}

func (u *URL) Parse() error {
	// url.Parse can't parse urls without prefixes, lets fix that
	var fixedurl string
	if strings.Index(u.RawUrl, "://") == -1 {
		fixedurl = "http://" + u.RawUrl
	} else {
		fixedurl = u.RawUrl
	}

	if parsed_url, err := url.Parse(fixedurl); err != nil {
		return err
	} else {
		// net/url/URL.Host is a 'host or host:port', lets split them
		if indx := strings.Index(parsed_url.Host, ":"); indx != -1 {
			u.Host = parsed_url.Host[:indx]
			u.Port = parsed_url.Host[indx+1:]
		} else {
			u.Host = parsed_url.Host
		}
		u.Path = parsed_url.Path
		// lets separate two-level domain from N-level subdomains (N>2)
		splitted := strings.SplitN(u.Host, ".", -1)
		if len(splitted) < 3 {
			u.Domain = u.Host
		} else {
			u.Domain = strings.Join(splitted[len(splitted)-2:len(splitted)], ".")
			u.SubDomain = strings.Join(splitted[:len(splitted)-2], ".")
		}
	}
	return nil
}
