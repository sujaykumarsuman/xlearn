// balloon touches 1 GiB of memory in 16 MiB chunks: past the case's memory.max it is
// OOM-killed inside its case cgroup (MLE), never the container (INV-14).
package main

import "os"

func main() {
	var keep [][]byte
	for n := 0; n < 1<<30; n += 16 << 20 {
		b := make([]byte, 16<<20)
		for i := 0; i < len(b); i += 4096 {
			b[i] = 1
		}
		keep = append(keep, b)
	}
	os.NewFile(4, "result").Write([]byte("survived"))
	_ = keep
}
