package libredirector

import (
	"bufio"
	"fmt"
	"os"
)

type Category struct {
	Title    string
	UrlsFile string
	PcreFile string
	urls     []URL
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
			url.Parse()
			fmt.Printf("%#v\n", url)
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}
	return nil
}
