package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeLintFixture(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeLintDocument(t *testing.T, document any) string {
	t.Helper()
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return writeLintFixture(t, "openrpc.json", string(data))
}
