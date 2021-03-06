package main

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"

	"restic"
	"restic/errors"
	"restic/repository"
)

var cmdLs = &cobra.Command{
	Use:   "ls [flags] [snapshot-ID ...]",
	Short: "list files in a snapshot",
	Long: `
The "ls" command allows listing files and directories in a snapshot.

The special snapshot-ID "latest" can be used to list files and directories of the latest snapshot in the repository.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLs(lsOptions, globalOptions, args)
	},
}

// LsOptions collects all options for the ls command.
type LsOptions struct {
	ListLong bool
	Host     string
	Tags     []string
	Paths    []string
}

var lsOptions LsOptions

func init() {
	cmdRoot.AddCommand(cmdLs)

	flags := cmdLs.Flags()
	flags.BoolVarP(&lsOptions.ListLong, "long", "l", false, "use a long listing format showing size and mode")

	flags.StringVarP(&lsOptions.Host, "host", "H", "", "only consider snapshots for this `host`, when no snapshot ID is given")
	flags.StringSliceVar(&lsOptions.Tags, "tag", nil, "only consider snapshots which include this `tag`, when no snapshot ID is given")
	flags.StringSliceVar(&lsOptions.Paths, "path", nil, "only consider snapshots which include this (absolute) `path`, when no snapshot ID is given")
}

func printTree(repo *repository.Repository, id *restic.ID, prefix string) error {
	tree, err := repo.LoadTree(context.TODO(), *id)
	if err != nil {
		return err
	}

	for _, entry := range tree.Nodes {
		Printf("%s\n", formatNode(prefix, entry, lsOptions.ListLong))

		if entry.Type == "dir" && entry.Subtree != nil {
			if err = printTree(repo, entry.Subtree, filepath.Join(prefix, entry.Name)); err != nil {
				return err
			}
		}
	}

	return nil
}

func runLs(opts LsOptions, gopts GlobalOptions, args []string) error {
	if len(args) == 0 && opts.Host == "" && len(opts.Tags) == 0 && len(opts.Paths) == 0 {
		return errors.Fatal("Invalid arguments, either give one or more snapshot IDs or set filters.")
	}

	repo, err := OpenRepository(gopts)
	if err != nil {
		return err
	}

	if err = repo.LoadIndex(context.TODO()); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(gopts.ctx)
	defer cancel()
	for sn := range FindFilteredSnapshots(ctx, repo, opts.Host, opts.Tags, opts.Paths, args) {
		Verbosef("snapshot %s of %v at %s):\n", sn.ID().Str(), sn.Paths, sn.Time)

		if err = printTree(repo, sn.Tree, string(filepath.Separator)); err != nil {
			return err
		}
	}
	return nil
}
