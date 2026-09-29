//go:build runner_it

package main

// The runner-it lane's test profiles (testgo@0, testgo-open@0). A release build (no runner_it
// tag) never links them; internal/runner/imports_test.go guards that.
import _ "github.com/sujaykumarsuman/xlearn/internal/runner/profile/testgo"
