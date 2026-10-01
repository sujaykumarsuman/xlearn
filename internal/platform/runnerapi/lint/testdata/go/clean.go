// want:
//go:build linux

package main

import (
	"container/heap"
	"fmt"
	"math/rand/v2"
	"sort"
	"unicode"
	"unicode/utf8"
)

// a //go:embed mention inside a plain comment is fine: it doesn't start the comment.
var _ = heap.Init
var _ = fmt.Sprint
var _ = rand.IntN
var _ = sort.Ints
var _ = unicode.IsLetter
var _ = utf8.RuneLen

func pairSum(nums []int, target int) []int { return nil }
