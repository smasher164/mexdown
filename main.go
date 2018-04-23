package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/smasher164/mexdown/gen/html"
	"github.com/smasher164/mexdown/parse"
)

func main() {
	f := parse.Parse(os.Stdin)
	os.Stdin.Close()
	if len(f.Errors) > 0 {
		for _, e := range f.Errors {
			fmt.Println(e)
		}
		os.Exit(1)
	}
	// litter.Dump(f)
	if _, err := io.Copy(os.Stdout, &html.Genner{File: f}); err != nil {
		log.Fatalln(err)
	}
}
