//go:build linux && arm64

package seccomp

const (
	nativeName  = "arm64"
	nativeAudit = 0xC00000B7 // AUDIT_ARCH_AARCH64
	nativeX32   = false
)
