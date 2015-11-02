package libredirector

import (
	"fmt"
)

func Checker(id string, in chan *Input, out chan string) {
	fmt.Printf("Checker%v started\n", id)
	defer WG.Done()
	for input := range in {
		if err := input.Url.Parse(); err != nil {
			out <- fmt.Sprintf("ParseUrl error: %+v", err)
		} else {
			out <- fmt.Sprintf("%#v", input.Url)
		}
	}
	fmt.Printf("Checker%v closed\n", id)
}
