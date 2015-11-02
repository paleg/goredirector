package libredirector

import (
	"bufio"
	"fmt"
	"os"
	"sort"
)

type Category struct {
	Title    string
	UrlsFile string
	PcreFile string
	Urls     map[string][]URL
}

func (c *Category) Print() {
	fmt.Printf("%v (%v)\n", c.Title, len(c.Urls))
	fmt.Printf("  urls: %v \n", c.UrlsFile)
	fmt.Printf("  pcre: %v \n", c.PcreFile)
	var keys []string
	for k := range c.Urls {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		fmt.Printf("    %v: %v\n", i, c.Urls[k])
	}
}

func (c *Category) Load() error {
	fmt.Printf("Loading '%+v' urls from '%+v'\n", c.Title, c.UrlsFile)
	if file, err := os.Open(c.UrlsFile); err != nil {
		fmt.Printf("Failed to load '%+v' category: %+v\n", c.Title, err)
		return err
	} else {
		defer file.Close()
		defer fmt.Printf("Loaded '%+v' category\n", c.Title)

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
