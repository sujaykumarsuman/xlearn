package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum"
)

// publicContent is the public half, loaded with the curriculum service's own glob loader
// (strict decoding, validation, id and slug guards), so packlint sees exactly what the
// seed sees.
type publicContent struct {
	root      string // the content root (holds courses/)
	content   *curriculum.Content
	items     map[string]*publicItem
	ids       []string // sorted by course, then id
	concepts  map[string]map[string]bool
	manifests map[string]*course.Manifest
}

type publicItem struct {
	course   string
	resolved *course.ResolvedItem
	// dir is the item directory relative to the content root.
	dir string
}

// contentRoot resolves --public: the repo root (curriculum/courses exists) or a content
// root (courses/ exists).
func contentRoot(p string) (string, error) {
	for _, cand := range []string{filepath.Join(p, "curriculum"), p} {
		if fi, err := os.Stat(filepath.Join(cand, "courses")); err == nil && fi.IsDir() {
			return cand, nil
		}
	}
	return "", fmt.Errorf("--public %s: no courses/ (want the repo root or a content root holding courses/)", p)
}

func loadPublic(p string) (*publicContent, error) {
	root, err := contentRoot(p)
	if err != nil {
		return nil, err
	}
	c, err := curriculum.LoadContent(os.DirFS(root))
	if err != nil {
		return nil, fmt.Errorf("public content %s: %w", root, err)
	}
	pc := &publicContent{
		root: root, content: c, items: map[string]*publicItem{},
		concepts: map[string]map[string]bool{}, manifests: c.Manifests,
	}
	for _, cc := range c.Courses {
		set := map[string]bool{}
		for _, cn := range cc.Concepts {
			set[cn.Slug] = true
		}
		pc.concepts[cc.Slug] = set
		for i := range cc.Items {
			ri := &cc.Items[i]
			pc.items[ri.Item.ID] = &publicItem{
				course: cc.Slug, resolved: ri,
				dir: filepath.ToSlash(filepath.Join("courses", cc.Slug, "items", ri.Item.ID)),
			}
			pc.ids = append(pc.ids, ri.Item.ID)
		}
	}
	sort.SliceStable(pc.ids, func(i, j int) bool {
		a, b := pc.items[pc.ids[i]], pc.items[pc.ids[j]]
		if a.course != b.course {
			return a.course < b.course
		}
		return lessID(pc.ids[i], pc.ids[j])
	})
	return pc, nil
}

// lessID orders ids numerically when both are numbers (DSA), else lexically.
func lessID(a, b string) bool {
	if len(a) != len(b) && isDigits(a) && isDigits(b) {
		return len(a) < len(b)
	}
	return a < b
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// itemAt decodes an item's item.json as of a git ref (nil when it did not exist).
func (pc *publicContent) itemAt(ref string, pi *publicItem) (*course.Item, error) {
	prefix, err := gitOut(pc.root, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	spec := ref + ":" + strings.TrimSpace(prefix) + pi.dir + "/" + course.ItemFile
	out, err := gitOut(pc.root, "show", spec)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "exists on disk, but not in") {
			return nil, nil
		}
		return nil, err
	}
	return course.DecodeItem([]byte(out))
}

// changedSince reports whether anything under an item-relative directory changed since
// ref, in the working tree (tracked changes and new untracked files).
func (pc *publicContent) changedSince(ref string, pi *publicItem, sub string) (bool, error) {
	p := pi.dir + "/" + sub
	out, err := gitOut(pc.root, "diff", "--name-only", ref, "--", p)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(out) != "" {
		return true, nil
	}
	out, err = gitOut(pc.root, "ls-files", "--others", "--exclude-standard", "--", p)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// gitOut runs git in dir and returns stdout; the error carries stderr.
func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
