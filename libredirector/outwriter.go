package libredirector

import "fmt"

func OutWriter(out chan string) {
	defer WGMain.Done()
	for data := range out {
		fmt.Printf("%#v\n", data)
	}
}
