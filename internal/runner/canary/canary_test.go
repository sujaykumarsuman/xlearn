package canary

import (
	"bytes"
	"testing"
)

func TestRunIsDeterministic(t *testing.T) {
	if testing.Short() {
		t.Skip("does ~150 ms of work")
	}
	var a, b bytes.Buffer
	if err := Run(&a); err != nil {
		t.Fatal(err)
	}
	if err := Run(&b); err != nil {
		t.Fatal(err)
	}
	if a.Len() != 8 || !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Errorf("checksums %x vs %x", a.Bytes(), b.Bytes())
	}
}
