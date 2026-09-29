package runner

import (
	"encoding/json"
	"fmt"
	"io"
)

// Manifest is what the spawner hands the front at spawn time, over a pipe on the front's fd 4
// (never argv or env): the bearer token (read by the spawner, which can read a 0400 Secret
// file) and the facts the spawner measured at startup. It holds no learner data.
type Manifest struct {
	Token     string            `json:"token"`
	Version   string            `json:"version"`
	BootEpoch string            `json:"boot_epoch"`
	CPUModel  string            `json:"cpu_model"`
	Profiles  []ProfileManifest `json:"profiles"`
}

// ProfileManifest is one profile's startup facts.
type ProfileManifest struct {
	Name          string `json:"name"`
	Toolchain     string `json:"toolchain"`
	ProfileSHA256 string `json:"profile_sha256"`
	BaselineBytes int64  `json:"baseline_bytes"`
}

// MaxManifestBytes bounds the manifest pipe read.
const MaxManifestBytes = 64 << 10

// ReadManifest reads and decodes the manifest (strict).
func ReadManifest(r io.Reader) (*Manifest, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxManifestBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxManifestBytes {
		return nil, fmt.Errorf("manifest over %d bytes", MaxManifestBytes)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if m.Token == "" || m.BootEpoch == "" {
		return nil, fmt.Errorf("manifest: missing token or boot epoch")
	}
	return &m, nil
}
