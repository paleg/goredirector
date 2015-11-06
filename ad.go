package main

import (
	"github.com/paleg/libadclient"
	"reflect"
	"strings"
)

func ExtendFromAD(list []string) (result []string) {
	for _, s := range list {
		if strings.HasPrefix(s, "ad:") && len(s) > 3 {
			group := s[3:]
			if users, err := adclient.GetUsersInGroup(group); err != nil {
				ErrorLogger.Printf("Failed to get AD group '%v' members: %v", group, err)
			} else {
				for _, user := range users {
					result = append(result, strings.ToLower(user))
				}
			}
		}
	}
	return
}

func (c *Config) ReloadAD(sync bool) {
	WGAD.Wait()
	WGAD.Add(1)
	defer WGAD.Done()
	adclient.New()
	defer adclient.Delete()
	adclient.Timelimit = 60
	adclient.Nettimeout = 60
	if err := adclient.Login(c.ADServer, c.ADUser, c.ADPassword, c.ADSearchBase); err != nil {
		ErrorLogger.Printf("Failed to AD login: %v", err)
		return
	}
	defer ErrorLogger.Printf("Reloaded AD groups\n")

	workid := ExtendFromFile(c.work_id)
	workid = append(workid, ExtendFromAD(c.work_id)...)
	if !reflect.DeepEqual(c.WorkID, workid) {
		c.WorkID = workid
		ErrorLogger.Printf("extended c.WorkID to %v", c.WorkID)
	}

	allowid := ExtendFromFile(c.allow_id)
	allowid = append(allowid, ExtendFromAD(c.work_id)...)
	if !reflect.DeepEqual(c.AllowID, allowid) {
		c.AllowID = allowid
		ErrorLogger.Printf("extended c.AllowID to %v", c.AllowID)
	}

	for _, cat := range c.Categories {
		cat_workid := ExtendFromFile(cat.work_id)
		cat_workid = append(cat_workid, ExtendFromAD(cat.work_id)...)
		if !reflect.DeepEqual(cat.WorkID, cat_workid) {
			cat.WorkID = cat_workid
			ErrorLogger.Printf("extended %v.WorkID to %v", cat.Title, cat.WorkID)
		}

		cat_allowid := ExtendFromFile(cat.allow_id)
		cat_allowid = append(cat_allowid, ExtendFromAD(cat.allow_id)...)
		if !reflect.DeepEqual(cat.AllowID, cat_allowid) {
			cat.AllowID = cat_allowid
			ErrorLogger.Printf("extended %v.AllowID to %v", cat.Title, cat.AllowID)
		}
	}
}
