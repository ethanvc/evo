package dgit

import (
	"context"
	"testing"
)

func TestCommand(t *testing.T) {
	ctx := context.Background()
	currentBranch, err := GetCurrentBranchName(ctx)
	_ = err
	_ = currentBranch
}
