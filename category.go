package main

import (
	"bufio"
	"net"
	"os"
	"regexp"
	"strings"
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

		c.Urls = make(map[string][]URL)
		for _, urls_file := range c.UrlsFiles {
			if file, err := os.Open(urls_file); err != nil {
				ErrorLogger.Printf("Failed to load urls for '%+v' category: %+v\n", c.Title, err)
			} else {
				defer file.Close()

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
	}
	return nil
}
