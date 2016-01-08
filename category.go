package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	ActionRedir int = iota
	ActionPass
)

type Category struct {
	Title     string
	UrlsFiles []string
	PcreFiles []string
	Urls      map[string][]URL
	Pcre      []*regexp.Regexp
	RedirUrl  string
	WorkIP    []*net.IPNet
	work_ip   []string
	AllowIP   []*net.IPNet
	allow_ip  []string
	WorkID    []string
	work_id   []string
	AllowID   []string
	allow_id  []string
	Log       bool
	Reverse   bool
	Action    int
	cache_dir *string
}

func (c *Category) CheckPCRE(inurl string) (bool, int) {
	for i, re := range c.Pcre {
		if re.MatchString(inurl) {
			return true, i + 1
		}
	}
	return false, 0
}

func (c *Category) CheckURL(inurl *URL) (bool, string) {
	if urls, ok := c.Urls[inurl.Domain]; !ok {
		return false, ""
	} else {
		if urls[0].Base() {
			return true, urls[0].String()
		} else {
			for _, url := range urls {
				if (url.SubDomain == "" || url.SubDomain == inurl.SubDomain || strings.HasSuffix(inurl.SubDomain, "."+url.SubDomain)) &&
					(url.Port == "" || url.Port == inurl.Port) &&
					(url.Dirs == "" || url.Dirs == inurl.Dirs || strings.HasPrefix(inurl.Dirs, url.Dirs+"/")) {
					return true, url.String()
				}
			}
		}
	}
	return false, ""
}

func (c *Category) Load() error {
	defer WGCategories.Done()

	if len(c.PcreFiles) != 0 {
		defer func() {
			ErrorLogger.Printf("Loaded '%+v' category (%v pcre)\n", c.Title, len(c.Pcre))
		}()

		for _, pcre_file := range c.PcreFiles {
			//ErrorLogger.Printf("Loading '%+v' pcre from '%+v'\n", c.Title, pcre_file)
			if file, err := os.Open(pcre_file); err != nil {
				ErrorLogger.Printf("Failed to load pcre for '%+v' category: %+v\n", c.Title, err)
			} else {
				defer file.Close()
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					re := scanner.Text()
					if compiled, err := regexp.Compile(re); err != nil {
						ErrorLogger.Printf("Failed to compile '%v': %v\n", re, err)
					} else {
						c.Pcre = append(c.Pcre, compiled)
					}
				}
			}
		}
	}

	//ErrorLogger.Printf("Loading '%+v' urls from '%+v'\n", c.Title, c.UrlsFile)
	if len(c.UrlsFiles) != 0 {
		defer func() {
			ErrorLogger.Printf("Loaded '%+v' category (%v domains)\n", c.Title, len(c.Urls))
		}()

		var RawFilesTime time.Time
		for _, urls_file := range c.UrlsFiles {
			info, err := os.Stat(urls_file)
			if err != nil {
				continue
			}
			if RawFilesTime.Before(info.ModTime()) {
				RawFilesTime = info.ModTime()
			}
		}

		var CachedFileTime time.Time
		if info, err := os.Stat(c.cachedFileName()); err == nil {
			CachedFileTime = info.ModTime()
		}

		if RawFilesTime.Before(CachedFileTime) {
			if err := c.loadCachedUrls(); err == nil {
				return nil
			} else {
				ErrorLogger.Printf("Failed to load cache from %v: %v", c.cachedFileName(), err)
			}
		}

		if err := c.loadRawUrls(); err != nil {
			return err
		} else {
			c.saveCachedUrls()
		}
	}
	return nil
}

func (c *Category) loadRawUrls() error {
	c.Urls = make(map[string][]URL)

	for _, urls_file := range c.UrlsFiles {
		if file, err := os.Open(urls_file); err != nil {
			ErrorLogger.Printf("Failed to load urls for '%+v' category: %+v\n", c.Title, err)
		} else {
			defer file.Close()
			ErrorLogger.Printf("Loading '%v' category from %v", c.Title, urls_file)

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				inurl := scanner.Text()
				if parsed_input_url, err := ParseUrl(inurl); err != nil {
					ErrorLogger.Printf("Wrong url in '%s' category: %v\n", c.Title, err)
					continue
				} else {
					if urls_in_map, ok := c.Urls[parsed_input_url.Domain]; ok {
						if urls_in_map[0].Base() {
							//fmt.Printf("removing %v because of %v\n", parsed_input_url, urls_in_map[0])
						} else if parsed_input_url.Base() {
							//fmt.Printf("replacing %v because of %v\n", urls_in_map, parsed_input_url)
							c.Urls[parsed_input_url.Domain] = []URL{parsed_input_url}
						} else {
							add := true
							for _, url_in_map := range urls_in_map {
								if url_in_map.EqualTo(parsed_input_url) {
									//fmt.Printf("removing dublicate %v\n", parsed_input_url)
									add = false
									break
								}
							}
							if add {
								c.Urls[parsed_input_url.Domain] = append(c.Urls[parsed_input_url.Domain], parsed_input_url)
							}
						}
					} else {
						c.Urls[parsed_input_url.Domain] = []URL{parsed_input_url}
					}
				}
			}
			if err := scanner.Err(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Category) cachedFileName() string {
	return fmt.Sprintf("%s/%s.cache", *c.cache_dir, c.Title)
}

func (c *Category) saveCachedUrls() {
	// http://stackoverflow.com/questions/20318445/generic-function-to-store-go-encoded-data
	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	if err := enc.Encode(c.Urls); err != nil {
		ErrorLogger.Printf("Failed to encode '%+v' category: %v", c.Title, err)
	} else {
		if err = ioutil.WriteFile(c.cachedFileName(), buff.Bytes(), 0600); err != nil {
			ErrorLogger.Printf("Failed to create file: %v", err)
		}
	}
}

func (c *Category) loadCachedUrls() error {
	ErrorLogger.Printf("Loading '%v' category from %v", c.Title, c.cachedFileName())
	n, err := ioutil.ReadFile(c.cachedFileName())
	if err != nil {
		return err
	}

	p := bytes.NewBuffer(n)
	dec := gob.NewDecoder(p)

	err = dec.Decode(&c.Urls)
	if err != nil {
		return err
	}
	return nil
}
