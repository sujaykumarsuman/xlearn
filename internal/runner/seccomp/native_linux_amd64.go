//go:build linux && amd64

package seccomp

const (
	nativeName  = "amd64"
	nativeAudit = 0xC000003E // AUDIT_ARCH_X86_64
	nativeX32   = true       // reject x32 numbers (nr ≥ 0x40000000), t3 §16.2
)
