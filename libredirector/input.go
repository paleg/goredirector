package libredirector

import (
	"errors"
	//"fmt"
	"net/url"
	"strings"
)

type URL struct {
	url.URL
	RawUrl    string
	Domain    string
	SubDomain string
}

type Input struct {
	Chanid string
	Url    URL
	IP     string
	User   string
	Method string
}

func (u *URL) Parse() error {
	if parsed_url, err := url.Parse(u.RawUrl); err != nil {
		return err
	} else {
		u.URL = *parsed_url
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

func ParseInput(input string) (i Input, err error) {
	splitted := strings.Split(input, " ")
	if len(splitted) < 4 {
		err = errors.New("Wrong number of arguments")
		return
	} else if len(splitted) == 4 {
		splitted = append([]string{"0"}, splitted...)
	}
	i.Chanid = splitted[0]
	// TODO: check err_url
	// TODO: CASE_INDEPENDENT
	// TODO: raw_change()
	i.Url.RawUrl = splitted[1]
	i.IP = splitted[2]
	// TODO: check err_user
	i.User, _ = url.QueryUnescape(splitted[3])
	i.User = strings.ToLower(i.User)
	i.Method = splitted[4]
	return
}
