package libredirector

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type URL struct {
	Domain    string
	SubDomain string
	Port      string
	Dirs      string
	Host      string
	RawUrl    string
}

type URLs []URL

func (u URLs) Len() int      { return len(u) }
func (u URLs) Swap(i, j int) { u[i], u[j] = u[j], u[i] }
func (u URLs) Less(i, j int) bool {
	return u[i].Domain+u[i].SubDomain+u[i].Port+u[i].Dirs <
		u[j].Domain+u[j].SubDomain+u[j].Port+u[j].Dirs
}

func (u *URLs) Merge() {
	indx2del := []int{}
	var base URL
	for i := 0; i < len(*u)-1; i++ {
		//fmt.Printf("comparing %v with %v\n", (*u)[i], (*u)[i+1])
		if (*u)[i].EqualTo(&(*u)[i+1]) {
			fmt.Printf("removing dublicate %v %v\n", i, (*u)[i])
			indx2del = append(indx2del, i)
			continue
		}
		if (*u)[i].Base() {
			base = (*u)[i]
			fmt.Printf("changing base to %v %v\n", i, base)
			i++
		}
		if base.Domain == (*u)[i].Domain {
			fmt.Printf("removing %v %v because of %v\n", i, (*u)[i], base)
			indx2del = append(indx2del, i)
		}
	}
	fmt.Printf("index2del = %v\n", indx2del)
	for i := len(indx2del); i > 0; i-- {
		indx := indx2del[i-1]
		*u = append((*u)[:indx], (*u)[indx+1:]...)
	}
}

func (u URL) Base() bool {
	return u.SubDomain == "" &&
		u.Port == "" &&
		u.Dirs == ""
}

func (u URL) String() string {
	return fmt.Sprintf("%v!%v!%v!%v", u.Domain, u.SubDomain, u.Port, u.Dirs)
}

func (u1 *URL) EqualTo(u2 *URL) bool {
	return u1.Domain == u2.Domain &&
		u1.SubDomain == u2.SubDomain &&
		u1.Port == u2.Port &&
		u1.Dirs == u2.Dirs
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
		for _, c := range u.Host {
			if !((c > 44 && c < 47) || (c > 47 && c < 58) || (c > 64 && c < 91) || c == 95 || (c > 96 && c < 123)) {
				return errors.New(fmt.Sprintf("Skipping %v because of bad character in domain name", u.Host))
			}
		}
		u.Dirs = parsed_url.Path
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
