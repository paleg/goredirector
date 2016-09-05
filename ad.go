package main

import (
	"github.com/paleg/libadclient"
	"reflect"
	"sort"
	"strings"
	"time"
)

func (c *Config) ExtendFromAD(binded_domain string, list []string) (result []string) {
	for _, s := range list {
		if strings.HasPrefix(s, "ad:") && len(s) > 3 {
			var group, domain string
			groupAtDomain := strings.Split(s[3:], "@")
			if len(groupAtDomain) == 1 {
				group = groupAtDomain[0]
				domain = c.ADDefaultDomain
			} else {
				group = groupAtDomain[0]
				domain = strings.ToUpper(groupAtDomain[1])
			}
			if domain != binded_domain {
				continue
			}

			if users, err := adclient.GetUsersInGroup(group, true); err != nil {
				ErrorLogger.Printf("Failed to get AD group '%v' members: %v", group, err)
			} else {
				for _, user := range users {
					result = append(result, strings.ToLower(user)+"@"+domain)
				}
			}
		}
	}
	return
}

func (c *Config) UseAD() bool {
	return c.ADDefaultDomain != "" &&
		c.ADReload != 0
	return false
}

func (c *Config) ReloadAD(sync bool) {
	if !c.UseAD() {
		return
	}

	if sync {
		c.ReloadADSync()
	} else {
		go c.ReloadADSync()
	}
}

func (c *Config) ReloadADSync() {
	if !c.UseAD() {
		return
	}

	WGAD.Wait()
	WGAD.Add(1)
	defer WGAD.Done()
	defer ErrorLogger.Printf("Reloaded AD groups\n")

	workid := c.ExtendFromFile(c.work_id)
	allowid := c.ExtendFromFile(c.allow_id)

	cat_workids := make(map[string][]string)
	cat_allowids := make(map[string][]string)
	for _, cat := range c.Categories {
		cat_workids[cat.Title] = c.ExtendFromFile(cat.work_id)
		cat_allowids[cat.Title] = c.ExtendFromFile(cat.allow_id)
	}

	for domain, settings := range c.ADDomains {
		params := adclient.DefaultADConnParams()
		params.Domain = domain
		//params.Site =
		params.Binddn = settings.Username
		params.Bindpw = settings.Password
		params.Search_base = settings.SearchBase
		params.Timelimit = 60
		params.Nettimeout = 60

		adclient.New()
		if err := adclient.Login(params); err != nil {
			ErrorLogger.Printf("Failed to AD login: %v", err)
			return
		}
		ErrorLogger.Printf("Logged on to %v", adclient.BindedUri())

		workid = append(workid, c.ExtendFromAD(domain, c.work_id)...)
		allowid = append(allowid, c.ExtendFromAD(domain, c.work_id)...)

		for _, cat := range c.Categories {
			cat_workids[cat.Title] = append(cat_workids[cat.Title], c.ExtendFromAD(domain, cat.work_id)...)
			cat_allowids[cat.Title] = append(cat_allowids[cat.Title], c.ExtendFromAD(domain, cat.allow_id)...)
		}
		adclient.Delete()
	}

	sort.Strings(workid)
	if !reflect.DeepEqual(c.WorkID, workid) {
		c.WorkID = workid
		ErrorLogger.Printf("extended c.WorkID to %v", c.WorkID)
	}

	sort.Strings(allowid)
	if !reflect.DeepEqual(c.AllowID, allowid) {
		c.AllowID = allowid
		ErrorLogger.Printf("extended c.AllowID to %v", c.AllowID)
	}

	for _, cat := range c.Categories {
		sort.Strings(cat_workids[cat.Title])
		sort.Strings(cat_allowids[cat.Title])

		if !reflect.DeepEqual(cat.WorkID, cat_workids[cat.Title]) {
			cat.WorkID = cat_workids[cat.Title]
			ErrorLogger.Printf("extended %v.WorkID to %v", cat.Title, cat.WorkID)
		}
		if !reflect.DeepEqual(cat.AllowID, cat_allowids[cat.Title]) {
			cat.AllowID = cat_allowids[cat.Title]
			ErrorLogger.Printf("extended %v.AllowID to %v", cat.Title, cat.AllowID)
		}
	}
}

func (c *Config) ScheduleReloadAD(seconds int) {
	if !c.UseAD() {
		return
	}

	ErrorLogger.Printf("Scheduling AD reload every %v second", seconds)
	c.ADTicker = time.NewTicker(time.Duration(seconds) * time.Second)
	c.ADTickerQuit = make(chan struct{})
	go func(cfg *Config) {
		for {
			select {
			case <-c.ADTicker.C:
				ErrorLogger.Printf("Reloading AD on schedule\n")
				cfg.ReloadADSync()
			case <-c.ADTickerQuit:
				ErrorLogger.Printf("Stopped AD reloading on schedule\n")
				c.ADTicker.Stop()
				return
			}
		}
	}(c)
}

func (c *Config) StopReloadAD() {
	if !c.UseAD() {
		return
	}

	close(c.ADTickerQuit)
}
