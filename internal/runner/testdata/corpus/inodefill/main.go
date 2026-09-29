// inodefill creates empty files in /w until the tmpfs runs out of inodes: the program's own
// error (exit 3), never an infra error.
package main

import (
	"fmt"
	"os"
)

func main() {
	for i := 0; ; i++ {
		f, err := os.Create(fmt.Sprintf("/w/i%d", i))
		if err != nil {
			fmt.Println("create:", err, "after", i)
			os.Exit(3)
		}
		f.Close()
	}
}
