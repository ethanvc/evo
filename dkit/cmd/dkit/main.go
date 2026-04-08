package main

import (
	"context"
	"os"

	"github.com/ethanvc/evo/dkit"
	"github.com/ethanvc/evo/xobs"
	"github.com/spf13/cobra"
)

// GOOS=linux GOARCH=amd64 go build -o dkit main.go
// GOBIN=$(pwd) go install github.com/ethanvc/evo/dkit/cmd/dkit@latest
// GOPROXY=direct GOBIN=$(pwd) go install github.com/ethanvc/evo/dkit/cmd/dkit@latest
func main() {
	ctx := context.Background()
	xobs.LogInfo(ctx, "dkit start")
	rootCmd := &cobra.Command{
		Use:          "dkit",
		Short:        "dkit",
		Long:         `dkit`,
		SilenceUsage: true,
	}
	dkit.AddDeleteMergedBranchesCmd(rootCmd)
	dkit.AddGrpcCmd(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
