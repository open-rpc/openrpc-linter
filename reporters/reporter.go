package reporters

import (
	"io"

	"github.com/open-rpc/openrpc-linter/types"
)

type Reporter interface {
	Format(results []types.RuleFunctionResult, totalRules int, output io.Writer) error
}
