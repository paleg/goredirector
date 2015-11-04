package libredirector

import "fmt"

func OutWriter(out chan string) {
	fmt.Printf("OutWriter started\n")
	defer WGMain.Done()
	for data := range out {
		fmt.Printf("%#v\n", data)
	}
	fmt.Printf("OutWriter closed\n")
}
