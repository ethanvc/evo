package dgit

import (
	"bytes"
	"context"
	"os/exec"
	"slices"
	"strings"

	"github.com/ethanvc/study/golangproj/xobs"
	"google.golang.org/grpc/codes"
)

func GetCurrentBranchName(ctx context.Context) (string, error) {
	buf, code, _, err := runCommand(ctx, "git", "branch", "--show-current")
	if err != nil || code != 0 {
		return "", xobs.New(codes.Unknown, "").SetMsgf("code:%v, err:%v, content:%s", code, err, buf)
	}
	return strings.TrimSpace(buf), nil
}

func ListAllRemoteBranches(ctx context.Context) ([]string, error) {
	buf, code, cmdLine, err := runCommand(ctx, "git", "branch", "-r")
	if err != nil {
		return nil, xobs.New(codes.Unknown, "").SetMsgf("code: %v, err: %v, cmd line: %s, content: %s",
			code, err, cmdLine, buf)
	}
	branches := splitStringByNewLine(buf)
	return branches, nil
}

func IsRemoteBranchExist(ctx context.Context, remoteBranch string) (bool, error) {
	branches, err := ListAllRemoteBranches(ctx)
	if err != nil {
		return false, err
	}
	return slices.Contains(branches, remoteBranch), nil
}

func ListMergedBranches(c context.Context, targetBranch string) ([]string, error) {
	buf, code, cmdLine, err := runCommand(c, "git", "branch", "--merged",
		targetBranch, `--format=%(refname:short)`)
	if err != nil {
		return nil, xobs.New(codes.Unknown, "").SetMsgf("code: %v, err: %v, cmd line: %s, content: %s",
			code, err, cmdLine, buf)
	}
	return splitStringByNewLine(buf), nil
}

func DeleteBranch(c context.Context, branchName string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, _, _, err := runCommand(c, "git", "branch", flag, branchName)
	if err != nil {
		return err
	}
	return nil
}

func runCommand(c context.Context, name string, args ...string) (string, int, string, error) {
	cmd := exec.CommandContext(c, name, args...)
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	cmd.Stderr = buf
	err := cmd.Run()
	return buf.String(), cmd.ProcessState.ExitCode(), cmd.String(), err
}

func splitStringByNewLine(s string) []string {
	parts := strings.Split(s, "\n")
	resultI := 0
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parts[resultI] = part
		resultI++
	}
	return parts[:resultI]
}
