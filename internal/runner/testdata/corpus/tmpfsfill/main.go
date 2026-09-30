// tmpfsfill writes 1 MiB files into /w until the per-case tmpfs is full: the program's own
// error (exit 3), never an infra error.
package main

import (
	"fmt"
	"os"
)

func main() {
	buf := make([]byte, 1<<20)
	for i := 0; ; i++ {
		f, err := os.Create(fmt.Sprintf("/w/f%d", i))
		if err != nil {
			fmt.Println("create:", err)
			os.Exit(3)
		}
		_, err = f.Write(buf)
		f.Close()
		if err != nil {
			fmt.Println("write:", err)
			os.Exit(3)
		}
	}
}
