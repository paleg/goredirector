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
	Urls     URLs
}

func (c *Category) Print() {
	fmt.Printf("%v (%v)\n", c.Title, len(c.Urls))
	fmt.Printf("  urls: %v \n", c.UrlsFile)
	fmt.Printf("  pcre: %v \n", c.PcreFile)
	for i, url := range c.Urls {
		fmt.Printf("    %v: %v\n", i, url)
	}
}

func (c *Category) Load() error {
	fmt.Printf("Loading '%+v' URLs from '%+v'\n", c.Title, c.UrlsFile)
	if file, err := os.Open(c.UrlsFile); err != nil {
		return err
	} else {
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var url URL
			url.RawUrl = scanner.Text()
			if err := url.Parse(); err != nil {
				fmt.Println(err)
				continue
			} else {
				c.Urls = append(c.Urls, url)
				fmt.Printf("%#v\n", url)
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		c.Print()
		sort.Sort(c.Urls)
		c.Print()
		c.Urls.Merge()
		c.Print()
		//c.Urls.
	}
	return nil
}
