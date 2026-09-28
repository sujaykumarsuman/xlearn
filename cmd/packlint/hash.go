package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
)

// runHash prints `<id> <content_hash> <contract_hash>` per public item ("-" for a
// self-path item's empty contract). The author pastes the contract hash into the pack's
// accepts_contract_hashes.
func runHash(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("packlint hash", flag.ContinueOnError)
	fl.SetOutput(stderr)
	public := fl.String("public", ".", "the public repo root, or a content root holding courses/")
	var items multiFlag
	fl.Var(&items, "item", "print only this item id (repeatable)")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(stderr, "packlint hash: unexpected arguments %v\n", fl.Args())
		return exitUsage
	}
	pub, err := loadPublic(*public)
	if err != nil {
		fmt.Fprintln(stderr, "packlint hash:", err)
		return exitUsage
	}
	only := map[string]bool{}
	for _, id := range items {
		if pub.items[id] == nil {
			fmt.Fprintf(stderr, "packlint hash: no public item %q\n", id)
			return exitUsage
		}
		only[id] = true
	}
	for _, id := range pub.ids {
		if len(only) > 0 && !only[id] {
			continue
		}
		ri := pub.items[id].resolved
		content, err := canon.ContentHash(ri)
		if err != nil {
			fmt.Fprintf(stderr, "packlint hash: item %s: %v\n", id, err)
			return exitFail
		}
		contract, err := canon.ContractHash(&ri.Item)
		if err != nil {
			fmt.Fprintf(stderr, "packlint hash: item %s: %v\n", id, err)
			return exitFail
		}
		if contract == "" {
			contract = "-"
		}
		fmt.Fprintf(stdout, "%s %s %s\n", id, content, contract)
	}
	return exitOK
}
