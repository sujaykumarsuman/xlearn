// sleep does nothing for an hour: the idle kill (CPU rate < 5% for ≥ 1 s) gives TLE (idle).
package main

import "time"

func main() {
	time.Sleep(time.Hour)
}
