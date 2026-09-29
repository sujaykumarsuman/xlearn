// threadbomb pins goroutines to OS threads until the Go runtime can't create another thread
// under pids.max (the runtime dies with "failed to create new OS thread": RE, fork limit).
package main

import (
	"runtime"
	"time"
)

func main() {
	for i := 0; i < 5000; i++ {
		go func() {
			runtime.LockOSThread()
			time.Sleep(time.Hour)
		}()
	}
	time.Sleep(time.Hour)
}
