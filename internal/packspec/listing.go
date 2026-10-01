package packspec

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// The image allowlist (t1 §3.3, the listing test): a built pack image may contain ONLY
//
//	/manifest.json
//	/courses/<slug>/items/<id>/cases.jsonl.zst
//	/courses/<slug>/items/<id>/{keys,anchors,exemplars}/**
//
// (and the directories leading to them), in one layer per course — every file of a course
// in its own layer, courses in sorted order — plus a last layer holding only the manifest,
// whose file hashes must verify against the image's files. Anything else — gen/, validate/,
// invalid/, submissions/, tests/, pack.json, tests.lock, timing.json, drafts/, a dotfile —
// fails: no generators, oracles, wrong solutions or calibration data ever ship.

// Listing is an image's (or a build directory's) files by layer.
type Listing struct {
	// Layers hold each layer's regular files ("/"-less paths, e.g. "manifest.json").
	Layers [][]string
	// Read returns a file's bytes from the final filesystem.
	Read func(p string) ([]byte, error)
}

var builtPathRe = regexp.MustCompile(`^courses/([a-z0-9]+(?:-[a-z0-9]+)*)/items/([A-Za-z0-9-]+)/(.+)$`)

// CheckListing applies the allowlist, the layer rule and the manifest verification. It
// returns every problem (paths only, never contents).
func CheckListing(l *Listing) []error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	all := map[string]bool{}
	layerCourse := make([]string, len(l.Layers))
	for i, layer := range l.Layers {
		for _, p := range layer {
			all[p] = true
			if p == ManifestFile {
				layerCourse[i] = "<manifest>"
				continue
			}
			sm := builtPathRe.FindStringSubmatch(p)
			if sm == nil || !AllowedItemFile(sm[3]) || path.Clean(p) != p {
				add("%s: not allowed in a pack image", p)
				continue
			}
			switch {
			case layerCourse[i] == "":
				layerCourse[i] = sm[1]
			case layerCourse[i] != sm[1]:
				add("layer %d mixes courses %s and %s (one layer per course)", i+1, layerCourse[i], sm[1])
			}
		}
	}
	if !all[ManifestFile] {
		add("no /%s", ManifestFile)
		return errs
	}
	if n := len(l.Layers); n == 0 || len(l.Layers[n-1]) != 1 || l.Layers[n-1][0] != ManifestFile {
		add("the last layer must hold only /%s", ManifestFile)
	}
	seen := map[string]bool{}
	var order []string
	for i := 0; i < len(l.Layers)-1; i++ {
		c := layerCourse[i]
		switch {
		case c == "":
			add("layer %d is empty", i+1)
		case c == "<manifest>":
			add("layer %d: /%s belongs in the last layer only", i+1, ManifestFile)
		case seen[c]:
			add("course %s spans more than one layer", c)
		default:
			seen[c] = true
			order = append(order, c)
		}
	}
	if !sort.StringsAreSorted(order) {
		add("course layers are not in sorted order")
	}
	mb, err := l.Read(ManifestFile)
	if err != nil {
		add("read /%s: %v", ManifestFile, err)
		return errs
	}
	m, err := ReadManifest(bytes.NewReader(mb))
	if err != nil {
		add("%v", err)
		return errs
	}
	listed := map[string]bool{ManifestFile: true}
	for id, it := range m.Items {
		for p, h := range it.Files {
			listed[p] = true
			b, err := l.Read(p)
			if err != nil {
				add("item %s: %s is in the manifest but not in the image", id, p)
				continue
			}
			if FileHash(b) != h {
				add("item %s: %s does not match its manifest hash", id, p)
			}
		}
	}
	for p := range all {
		if !listed[p] {
			add("%s: in the image but not in the manifest", p)
		}
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}

// ListDir lists a build directory as an image would hold it: one layer per courses/<slug>/
// and the manifest, ignoring the generated Dockerfile at its root (it is the recipe, not
// content). Any other top-level entry is listed too, so it fails the allowlist.
func ListDir(dir string) (*Listing, error) {
	byCourse := map[string][]string{}
	var other []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if rel == "." || d.IsDir() {
			return nil
		}
		switch {
		case rel == DockerfileName:
		case rel == ManifestFile:
		case strings.HasPrefix(rel, "courses/") && strings.Count(rel, "/") >= 2:
			c := strings.Split(rel, "/")[1]
			byCourse[c] = append(byCourse[c], rel)
		default:
			other = append(other, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	l := &Listing{Read: func(p string) ([]byte, error) { return os.ReadFile(filepath.Join(dir, filepath.FromSlash(p))) }}
	if len(other) > 0 {
		sort.Strings(other)
		l.Layers = append(l.Layers, other)
	}
	for _, c := range sortedKeys(byCourse) {
		files := byCourse[c]
		sort.Strings(files)
		l.Layers = append(l.Layers, files)
	}
	l.Layers = append(l.Layers, []string{ManifestFile})
	return l, nil
}

// ListImageTar lists a `docker save` / buildx `type=docker` archive, or an OCI layout
// archive (`type=oci`): the image's layers in order (gzip, zstd or plain tar blobs).
func ListImageTar(archive string) (*Listing, error) {
	f, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	members := map[string][]byte{}
	tr := tar.NewReader(bufio.NewReader(f))
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", archive, err)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(tr, 1<<30))
		if err != nil {
			return nil, err
		}
		members[path.Clean(h.Name)] = b
	}
	layers, err := layerOrder(members)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", archive, err)
	}
	content := map[string][]byte{}
	l := &Listing{Read: func(p string) ([]byte, error) {
		b, ok := content[p]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return b, nil
	}}
	for _, name := range layers {
		blob, ok := members[name]
		if !ok {
			return nil, fmt.Errorf("%s: layer %s missing", archive, name)
		}
		r, err := decompressLayer(blob)
		if err != nil {
			return nil, fmt.Errorf("layer %s: %w", name, err)
		}
		lt := tar.NewReader(r)
		var files []string
		for {
			h, err := lt.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("layer %s: %w", name, err)
			}
			p := strings.TrimPrefix(path.Clean("/"+h.Name), "/")
			switch h.Typeflag {
			case tar.TypeDir:
				continue
			case tar.TypeReg:
				b, err := io.ReadAll(io.LimitReader(lt, 1<<30))
				if err != nil {
					return nil, err
				}
				content[p] = b
				files = append(files, p)
			default:
				files = append(files, p) // links, devices, whiteouts: listed, so they fail
			}
		}
		sort.Strings(files)
		l.Layers = append(l.Layers, files)
	}
	return l, nil
}

// layerOrder reads the archive's image manifest: docker-save's manifest.json, else the OCI
// index.json → manifest → layers.
func layerOrder(members map[string][]byte) ([]string, error) {
	if b, ok := members["manifest.json"]; ok {
		var dm []struct {
			Layers []string `json:"Layers"`
		}
		if err := json.Unmarshal(b, &dm); err != nil {
			return nil, fmt.Errorf("manifest.json: %w", err)
		}
		if len(dm) != 1 {
			return nil, fmt.Errorf("manifest.json lists %d images, want 1", len(dm))
		}
		return dm[0].Layers, nil
	}
	b, ok := members["index.json"]
	if !ok {
		return nil, errors.New("neither a docker-save manifest.json nor an OCI index.json")
	}
	type desc struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
	}
	var idx struct {
		Manifests []desc `json:"manifests"`
	}
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	blob := func(d string) ([]byte, error) {
		alg, hex, _ := strings.Cut(d, ":")
		b, ok := members[path.Join("blobs", alg, hex)]
		if !ok {
			return nil, fmt.Errorf("blob %s missing", d)
		}
		return b, nil
	}
	for len(idx.Manifests) == 1 && strings.Contains(idx.Manifests[0].MediaType, "index") {
		b, err := blob(idx.Manifests[0].Digest)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &idx); err != nil {
			return nil, err
		}
	}
	if len(idx.Manifests) == 0 {
		return nil, errors.New("index.json lists no manifest")
	}
	// A multi-arch index: every platform carries the same data layers; check the first.
	mb, err := blob(idx.Manifests[0].Digest)
	if err != nil {
		return nil, err
	}
	var man struct {
		Layers []desc `json:"layers"`
	}
	if err := json.Unmarshal(mb, &man); err != nil {
		return nil, err
	}
	var out []string
	for _, l := range man.Layers {
		alg, hex, _ := strings.Cut(l.Digest, ":")
		out = append(out, path.Join("blobs", alg, hex))
	}
	return out, nil
}

func decompressLayer(b []byte) (io.Reader, error) {
	switch {
	case len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b:
		return gzip.NewReader(bytes.NewReader(b))
	case len(b) >= 4 && bytes.Equal(b[:4], []byte{0x28, 0xb5, 0x2f, 0xfd}):
		d, err := zstd.NewReader(bytes.NewReader(b), zstd.WithDecoderConcurrency(1))
		if err != nil {
			return nil, err
		}
		return d.IOReadCloser(), nil
	}
	return bytes.NewReader(b), nil
}
