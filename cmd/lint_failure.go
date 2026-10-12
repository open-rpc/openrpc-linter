package cmd

import (
	"fmt"
	"io"
)

// lintFailure carries stage evidence to the command's output boundary.
// Callers still receive the original cause after the diagnostic is written.
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
	fmt.Fprintln(output, failure.Error())
	return failure.cause
}
