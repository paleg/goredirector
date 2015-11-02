package libredirector

import (
	"errors"
	//"fmt"
	"net/url"
	"strings"
)

type URL struct {
	url.URL
	Domain    string
	SubDomain string
}

type Input struct {
	Chanid string
	RawUrl string
	Url    URL
	IP     string
	User   string
	Method string
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
	i.RawUrl = splitted[1]
	i.IP = splitted[2]
	// TODO: check err_user
	i.User, _ = url.QueryUnescape(splitted[3])
	i.User = strings.ToLower(i.User)
	i.Method = splitted[4]
	return
}

func (i *Input) ParseUrl() (err error) {
	if parsed_url, perr := url.Parse(i.RawUrl); err != nil {
		err = perr
	} else {
		i.Url.URL = *parsed_url
		splitted := strings.SplitN(i.Url.Host, ".", -1)
		if len(splitted) < 3 {
			i.Url.Domain = i.Url.Host
		} else {
			i.Url.Domain = strings.Join(splitted[len(splitted)-2:len(splitted)], ".")
			i.Url.SubDomain = strings.Join(splitted[:len(splitted)-2], ".")
		}
	}
	return
}
