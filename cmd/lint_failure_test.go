package cmd

import (
	"bytes"
	"errors"
	"testing"
)

func TestReportLintFailurePreservesWriteError(t *testing.T) {
	cause := errors.New("invalid document")
	writeErr := errors.New("output unavailable")
	failure := &lintFailure{prefix: "Error: ", cause: cause}
	err := reportLintFailure(failure, failingWriter{err: writeErr})
	if !errors.Is(err, cause) || !errors.Is(err, writeErr) {
		t.Fatalf("expected both errors, got %v", err)
	}
}

func TestReportLintFailureReturnsCauseAfterWriting(t *testing.T) {
	cause := errors.New("invalid document")
	var output bytes.Buffer
	err := reportLintFailure(&lintFailure{prefix: "Error: ", cause: cause}, &output)
	if err != cause || output.String() != "Error: invalid document\n" {
		t.Fatalf("error=%v, output=%q", err, output.String())
	}
}
