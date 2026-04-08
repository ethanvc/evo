package dkit

import (
	"context"
	"fmt"

	"github.com/ethanvc/evo/dkit/dgit"
	"github.com/spf13/cobra"
)

func AddDeleteMergedBranchesCmd(rootCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use: "delete-merged-branches",
	}
	dryRunFlag := cmd.Flags().Bool("dry-run", false, "dry run")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return DeleteMergedBranches(&DeleteMergedBranchesReq{
			DryRun: *dryRunFlag,
		})
	}
	rootCmd.AddCommand(cmd)
}

type DeleteMergedBranchesReq struct {
	DryRun bool
}

func DeleteMergedBranches(req *DeleteMergedBranchesReq) error {
	ctx := context.Background()
	if req.DryRun {
		fmt.Printf("Notice: dry run mode\n")
	}
	currentBranch, err := dgit.GetCurrentBranchName(ctx)
	if err != nil {
		return err
	}

	for _, targetBranch := range productionBranches {
		exist, err := dgit.IsRemoteBranchExist(ctx, targetBranch)
		if err != nil {
			return err
		}
		if !exist {
			continue
		}
		branches, err := dgit.ListMergedBranches(ctx, targetBranch)
		if err != nil {
			return err
		}
		for _, branch := range branches {
			if branch == currentBranch {
				fmt.Println("skip delete current branch")
				continue
			}
			if req.DryRun {
				fmt.Printf("dry run mode, delete branch %s\n", branch)
			} else {
				err := dgit.DeleteBranch(ctx, branch, true)
				if err != nil {
					return err
				}
				fmt.Printf("delete branch %s\n", branch)
			}
		}
	}
	return nil
}

var productionBranches = []string{"origin/master", "origin/main", "origin/release"}
