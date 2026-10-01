// Command stores-a-key is fx-003's custom validator: the statement promises that every call
// sequence stores at least one key (a structural rule the constraint grammar cannot say).
// It reads one canonical input on stdin and exits 0 when valid, 1 when not.
package main

import (
	"encoding/json"
	"os"
)

func main() {
	var in struct {
		Ops []string `json:"ops"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		os.Exit(2)
	}
	for _, op := range in.Ops {
		if op == "Put" {
			os.Exit(0)
		}
	}
	os.Exit(1)
}
