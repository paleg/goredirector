package main

import (
	"errors"
	//"fmt"
	"net"
	"net/url"
	"strings"
)

type Input struct {
	Chanid string
	RawUrl string
	IP     net.IP
	Host   string
	User   string
	Method string
	Raw    string
}

func ParseInput(input string) (i Input, err error) {
	i.Raw = input
	splitted := strings.Split(input, " ")
	if len(splitted) < 4 {
		err = errors.New("Wrong number of arguments")
		return
	} else if len(splitted) == 4 {
		splitted = append([]string{"0"}, splitted...)
	}
	i.Chanid = splitted[0]
	// TODO: CASE_INDEPENDENT
	i.RawUrl = splitted[1]
	if indx := strings.Index(splitted[2], "/"); indx == -1 {
		i.IP = net.ParseIP(splitted[2])
	} else {
		i.IP = net.ParseIP(splitted[2][:indx])
		if config.LogHost {
			i.Host = splitted[2][indx+1:]
		}
	}
	if i.User, err = url.QueryUnescape(splitted[3]); err != nil {
		//TODO: check wtf
		ErrorLogger.Printf("Failed to unescape user ident: %v\n", err)
		i.User = splitted[3]
		err = nil
	}
	if !config.UseAD() {
		i.User = strings.ToLower(i.User)
	} else {
		if indx := strings.Index(i.User, "@"); indx == -1 {
			i.User = strings.ToLower(i.User) + "@" + config.ADDefaultDomain
		} else {
			userAtDomain := strings.Split(i.User, "@")
			i.User = strings.ToLower(userAtDomain[0]) + "@" + strings.ToUpper(userAtDomain[1])
		}
	}
	i.Method = splitted[4]
	return
}
