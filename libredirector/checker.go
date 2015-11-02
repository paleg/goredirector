package libredirector

import (
	"fmt"
)

func Checker(id string, in chan *Input, out chan string) {
	fmt.Printf("Checker%v started\n", id)
	defer WG.Done()
	for input := range in {
		if url, err := ParseUrl(input.RawUrl); err != nil {
			out <- fmt.Sprintf("ParseUrl error: %+v", err)
		} else {
			out <- fmt.Sprintf("%#v", url)
		}
	}
	fmt.Printf("Checker%v closed\n", id)
}
