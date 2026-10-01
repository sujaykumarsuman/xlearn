package front

import (
	"slices"

	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// panicFrameMax bounds the fd-4 head the front keeps for the panic check: the longest
// {"panic":"<class>"} frame is well under it.
const panicFrameMax = 64

// panicClass reports the class of a case's {"panic":"<class>"} fd-4 frame (CaseResult
// PanicClass, judge's RE detail), or "". Only a whole frame of exactly that shape with a class
// from the language's closed set counts; anything else is judge's to decode. The bytes are the
// learner process's, so this never decides more than a label: the term is measured outside it.
func panicClass(lang string, frame []byte) string {
	payload, err := harness.SplitFrame(frame)
	if err != nil {
		return ""
	}
	const pre, post = `{"panic":"`, `"}`
	s := string(payload)
	if len(s) <= len(pre)+len(post) || s[:len(pre)] != pre || s[len(s)-len(post):] != post {
		return ""
	}
	cls := s[len(pre) : len(s)-len(post)]
	if slices.Contains(harness.PanicClasses[lang], cls) {
		return cls
	}
	return ""
}
