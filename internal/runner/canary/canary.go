// Package canary is the runner-owned throttling canary (s2, t3 §5.7): ~150 ms of mixed ALU
// work plus a memory stream larger than the LLC (> 16 MiB per L3 slice on the EPYC 9355P, t3
// §15). The spawner runs it as `runner canary`, capless, in a slot cgroup, and reads its CPU
// time from the cgroup: cgroup CPU time includes steal and slows under memory-bandwidth
// contention, which is exactly what the canary is there to see. The work is fixed, so its CPU
// time is comparable run to run; the rolling median lives in the spawner.
package canary

import "io"

// Work sizes. A 48 MiB buffer is three L3 slices' worth; the ALU loop is sized for ~100 ms
// on the production CPU, the stream for ~50 ms.
const (
	StreamBytes = 48 << 20
	ALURounds   = 60_000_000
	Passes      = 2
)

// Run does the fixed work and writes an 8-byte checksum to w (so the compiler can't drop it).
func Run(w io.Writer) error {
	buf := make([]byte, StreamBytes)
	var acc uint64 = 0x9E3779B97F4A7C15
	for p := 0; p < Passes; p++ {
		// Memory stream: a strided write then read pass over the whole buffer.
		for i := 0; i < len(buf); i += 64 {
			buf[i] = byte(acc >> (i & 31))
		}
		for i := 0; i < len(buf); i += 64 {
			acc += uint64(buf[i])
		}
	}
	// ALU: xorshift + multiply + divide.
	x := acc | 1
	for i := 0; i < ALURounds; i++ {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		acc += x * 0x2545F4914F6CDD1D
		if i&1023 == 0 {
			acc /= (x & 0xff) | 1
		}
	}
	var out [8]byte
	for i := range out {
		out[i] = byte(acc >> (8 * i))
	}
	_, err := w.Write(out[:])
	return err
}
