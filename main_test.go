package main

import (
	"os"
	"testing"
)

func TestMainHelp(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"openrpc-linter", "--help"}
	main()
}
