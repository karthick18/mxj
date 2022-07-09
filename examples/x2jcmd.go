// Per: https://github.com/karthick18/mxj/issues/24
// Per: https://github.com/karthick18/mxj/issues/25

package main

import (
	"fmt"
	"io"
	"os"
	"github.com/karthick18/mxj/x2j"
)

func main() {
	for {
		_, _, err := x2j.XmlReaderToJsonWriter(os.Stdin, os.Stdout)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			break
		}
	}
}
