package runnerapi

import "fmt"

// The generic file-name rule (t3 §2.4 A5). Judge lints before enqueue; the runner's front
// re-checks it before writing a byte. Per-profile extension and directive rules are m3-04's
// (runnerapi/lint).
const (
	MaxFileNameBytes = 64      // ≤ 64 bytes per name
	MaxFiles         = 32      // ≤ 32 files (Files + HiddenFiles)
	MaxFilesBytes    = 1 << 20 // ≤ 1 MiB of file data in total
)

// ValidFileName reports whether name is a flat file name: one or more of [A-Za-z0-9_], one
// dot, one or more of [A-Za-z0-9_]. That excludes separators, "..", a leading dot, spaces,
// control bytes and anything non-ASCII.
func ValidFileName(name string) error {
	if name == "" {
		return fmt.Errorf("empty file name")
	}
	if len(name) > MaxFileNameBytes {
		return fmt.Errorf("file name is %d bytes, cap %d", len(name), MaxFileNameBytes)
	}
	dot := -1
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c == '.':
			if dot >= 0 {
				return fmt.Errorf("file name %q: more than one dot", name)
			}
			dot = i
		case isNameByte(c):
		default:
			return fmt.Errorf("file name %q: byte 0x%02x not allowed", name, c)
		}
	}
	if dot <= 0 || dot == len(name)-1 {
		return fmt.Errorf("file name %q: want <base>.<ext>", name)
	}
	return nil
}

func isNameByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

// ValidateFiles checks the names, uniqueness and totals over Files and HiddenFiles together.
func ValidateFiles(files, hidden []File) error {
	if len(files)+len(hidden) > MaxFiles {
		return fmt.Errorf("%d files, cap %d", len(files)+len(hidden), MaxFiles)
	}
	seen := make(map[string]bool, len(files)+len(hidden))
	total := 0
	for _, set := range [][]File{files, hidden} {
		for _, f := range set {
			if err := ValidFileName(f.Path); err != nil {
				return err
			}
			if seen[f.Path] {
				return fmt.Errorf("duplicate file name %q", f.Path)
			}
			seen[f.Path] = true
			total += len(f.Data)
			if total > MaxFilesBytes {
				return fmt.Errorf("file data exceeds %d bytes", MaxFilesBytes)
			}
		}
	}
	return nil
}
