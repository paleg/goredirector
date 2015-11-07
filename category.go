package main

import (
	"bufio"
	"net"
	"os"
	"regexp"
	"strings"
)

type Category struct {
	Title    string
	UrlsFile string
	PcreFile string
	Urls     map[string][]URL
	Pcre     []*regexp.Regexp
	RedirUrl string
	WorkIP   []*net.IPNet
	work_ip  []string
	AllowIP  []*net.IPNet
	allow_ip []string
	WorkID   []string
	work_id  []string
	AllowID  []string
	allow_id []string
	Log      bool
	Reverse  bool
}

func (c *Category) CheckURL(inurl *URL) bool {
	if urls, ok := c.Urls[inurl.Domain]; !ok {
		ErrorLogger.Printf("%v is not in %v Urls\n", inurl.Domain, c.Title)
		return false
	} else {
		if urls[0].Base() {
			ErrorLogger.Printf("inurl.Domain is a base domain in %v Urls\n", c.Title)
			return true
		} else {
			for _, url := range urls {
				if (url.SubDomain == "" || url.SubDomain == inurl.SubDomain) &&
					(url.Port == "" || url.Port == inurl.Port) &&
					(url.Dirs == "" || url.Dirs == inurl.Dirs ||
						(strings.HasPrefix(inurl.Dirs, url.Dirs) && inurl.Dirs[len(url.Dirs)] == 47)) {
					ErrorLogger.Printf("%v == %v\n", inurl, url)
					return true
				} else {
					ErrorLogger.Printf("%v != %v\n", inurl, url)
				}
			}
		}
	}
	return false
}

func (c *Category) Load() error {
	defer WGCategories.Done()

	if c.PcreFile != "" {
		//ErrorLogger.Printf("Loading '%+v' pcre from '%+v'\n", c.Title, c.PcreFile)
		if file, err := os.Open(c.PcreFile); err != nil {
			ErrorLogger.Printf("Failed to load pcre for '%+v' category: %+v\n", c.Title, err)
		} else {
			defer file.Close()
			//defer func() {
			//	ErrorLogger.Printf("Loaded '%+v' category (%v pcre)\n", c.Title, len(c.Pcre))
			//}()
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

	//ErrorLogger.Printf("Loading '%+v' urls from '%+v'\n", c.Title, c.UrlsFile)
	if file, err := os.Open(c.UrlsFile); err != nil {
		ErrorLogger.Printf("Failed to load urls for '%+v' category: %+v\n", c.Title, err)
	} else {
		defer file.Close()
		defer func() {
			ErrorLogger.Printf("Loaded '%+v' category (%v domains)\n", c.Title, len(c.Urls))
		}()

		c.Urls = make(map[string][]URL)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if parsed_input_url, err := ParseUrl(scanner.Text()); err != nil {
				//fmt.Println(err)
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
		//c.Print()
	}
	return nil
}
