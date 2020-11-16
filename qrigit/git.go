package qrigit

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/lib"
)

// GitImporter is a state machine for importing git files
type GitImporter struct {
	inst *lib.Instance
}

// NewGitImportor constructs a git importer
func NewGitImporter(ctx context.Context) (*GitImporter, error) {
	if _, err := findGitBinary(); err != nil {
		return nil, fmt.Errorf("your system doesn't seem to have git installed: %w", err)
	}

	inst, err := lib.NewInstance(ctx, standardRepoPath())
	if err != nil {
		return nil, err
	}

	return &GitImporter{
		inst: inst,
	}, nil
}

// ImportGitFile creates a dataset for a single body file within a git repo
// to a datset named dsname
func (gi *GitImporter) ImportGitFile(dsname, gitDir, bodyFile string) (*dataset.Dataset, error) {
	commits, err := listFileCommits(gitDir, bodyFile)
	if err != nil {
		return nil, err
	}

	dsm := lib.NewDatasetMethods(gi.inst)
	for _, cm := range commits {
		fileRevision, err := getFileAtCommit(gitDir, cm.Hash, bodyFile)
		if err != nil {
			return nil, err
		}

		fmt.Printf("importing %s %s\n", cm.Hash[:7], cm.Title)
		p := &lib.SaveParams{
			Ref: fmt.Sprintf("me/%s", dsname),
			Dataset: &dataset.Dataset{
				Commit: &dataset.Commit{
					Title: cm.Title,
				},
				BodyPath:  "body.csv",
				BodyBytes: fileRevision,
			},
			Replace: true,
		}
		res := &dataset.Dataset{}
		if err := dsm.Save(p, res); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func findGitBinary() (string, error) {
	buf := &bytes.Buffer{}
	cmd := exec.Command("which", "git")
	cmd.Stdout = buf
	err := cmd.Run()
	return buf.String(), err
}

type gitCommit struct {
	Hash  string
	Title string
}

func listFileCommits(dir, file string) ([]gitCommit, error) {
	// git log --reverse --format=oneline --follow --name-status --
	// git log --reverse --format='%H %T' --follow --name-status --
	// res, err := runGitCommand(dir, "log", "--reverse", "--format=oneline", "--follow", "--name-status", "--", file)
	res, err := runGitCommand(dir, "log", "--reverse", "--format=%H %s", "--follow", "--", file)
	if err != nil {
		return nil, err
	}
	commits := []gitCommit{}
	sc := bufio.NewScanner(bytes.NewBuffer(res))
	for sc.Scan() {
		cm := gitCommit{
			Hash:  string(sc.Bytes()[:40]),
			Title: string(sc.Bytes()[41:]),
		}
		commits = append(commits, cm)
	}
	fmt.Printf("%#v\n", commits)
	return commits, nil
}

func getFileAtCommit(dir, commit, file string) ([]byte, error) {
	return runGitCommand(dir, "show", fmt.Sprintf("%s:%s", commit, file))
}

func runGitCommand(dir, cmdName string, arg ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{cmdName}, arg...)...)
	cmd.Dir = dir
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = errBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf(errBuf.String())
	}

	return buf.Bytes(), nil
}

// standardRepoPath returns qri paths based on the QRI_PATH environment
// variable falling back to the default: $HOME/.qri
func standardRepoPath() string {
	qriRepoPath := os.Getenv("QRI_PATH")
	if qriRepoPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			panic(err)
		}
		qriRepoPath = filepath.Join(home, ".qri")
	}

	return qriRepoPath
}
