package main

import (
	"fmt"
	"os"

	"mcesar.io/ofx"
)

func main() {
	doc, err := ofx.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Err: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(doc)
}
