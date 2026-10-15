package cmd

import (
	"errors"
	"fmt"
	"io"
)

// lintFailure carries stage evidence to the command's output boundary.
// Callers receive the original cause and any diagnostic write error.
type lintFailure struct {
	prefix string
	cause  error
}

func (failure *lintFailure) Error() string {
	return failure.prefix + failure.cause.Error()
}

func reportLintFailure(err error, output io.Writer) error {
	failure, ok := err.(*lintFailure)
	if !ok {
		return err
	}
	if _, writeErr := fmt.Fprintln(output, failure.Error()); writeErr != nil {
		return errors.Join(failure.cause, writeErr)
	}
	return failure.cause
}
