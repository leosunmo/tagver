package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var getTag, getBranch, getCommit, getDefault, ignoreUncleanTag bool
var path string

// Basic example of how to list tags.
func main() {

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of tagver: [-t] [-b] [-c] [<git dir>]\n\n")
		fmt.Fprint(flag.CommandLine.Output(), "Default output is very close to \"git describe --tags\":\n")
		fmt.Fprint(flag.CommandLine.Output(), "\tIf HEAD is not tagged: <tag>-<commits since tag>-<HEAD SHA> (example: v1.0.4-1-5227b593)\n")
		fmt.Fprint(flag.CommandLine.Output(), "\tIf HEAD is tagged: <tag> (example: v1.0.5)\n\n")
		fmt.Fprint(flag.CommandLine.Output(), "If \"-b\" or \"-c\" are provided with \"-t\", only the tag name will print regardless if it's clean or not.\n")
		fmt.Fprint(flag.CommandLine.Output(), "Print order will be <tag>-<branch>-<SHA>.\n\n")
		fmt.Fprintln(flag.CommandLine.Output(), "Set one or more flags.")

		flag.PrintDefaults()
	}

	flag.BoolVar(&getTag, "t", false, "Return the latest semver tag (annotated or lightweight Git tag) (default)")
	flag.BoolVar(&getBranch, "b", false, "Return the current branch")
	flag.BoolVar(&getCommit, "c", false, "Return the current commit")
	flag.BoolVar(&ignoreUncleanTag, "ignore-unclean-tag", false, "Return only tag name even if the latest tag doesn't point to HEAD (\"v1.0.4\" instead of \"v1.0.4-1-89c22b28\")")

	flag.Parse()

	if !getTag && !getBranch && !getCommit {
		getDefault = true
	}

	if args := flag.Args(); len(args) != 0 {
		path = args[0]
	} else {
		path = "./"
	}

	// We instance a new repository targeting the given path (the .git folder)
	r, err := plainOpen(path)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			log.Fatalf("Directory %s is not a git repository", path)
		} else {
			log.Fatalf("Failed to open git repo %s", path)
		}
	}

	var commit, branch, tag string

	if getTag || getDefault {
		var err error
		var count int
		var hash string
		tag, count, hash, err = getLatestTagFromRepository(r)
		if err != nil {
			log.Fatalf("Failed to get latest tag from %s, err %s\n", path, err.Error())
		}
		if count != 0 && !ignoreUncleanTag && !getBranch && !getCommit {
			tag = fmt.Sprintf("%v-%v-%v",
				tag,
				count,
				hash)
		}
		if tag == "" && getDefault {
			getBranch = true
			getCommit = true
		}
	}

	if getBranch {
		var err error
		// if in detached head, get branch from commit
		hr, err := r.Head()
		if err != nil {
			log.Fatalf("Failed to get head from %s, err %s\n", path, err.Error())
		}
		if hr.Name() == plumbing.HEAD {
			branch, err = getCurrentBranchFromDetachedHead(r)
		} else {
			branch, err = getCurrentBranchFromRepository(r)
		}
		if err != nil {
			log.Fatalf("Failed to get current branch from %s, err %s\n", path, err.Error())
		}
	}

	if getCommit {
		var err error
		commit, err = getCurrentCommitFromRepository(r)
		if err != nil {
			log.Fatalf("Failed to get current commit from %s, err %s\n", path, err.Error())
		}
	}

	pt := []string{tag, branch, commit}
	var versionInfo []string
	for _, s := range pt {
		if s != "" {
			versionInfo = append(versionInfo, s)
		}
	}
	if len(versionInfo) == 0 {
		log.Println("no version information found")
		return
	}
	fmt.Printf("%s\n", strings.Join(versionInfo, "-"))
}

func getCurrentBranchFromDetachedHead(r *Git) (string, error) {
	commit, err := r.Head()
	if err != nil {
		return "", err
	}

	memo := make(map[plumbing.Hash]bool)

	rs, err := r.References()

	if err != nil {
		return "", err
	}

	var branches []plumbing.ReferenceName

	rs.ForEach(func(ref *plumbing.Reference) error {
		n := ref.Name()
		if n.IsBranch() || n.IsRemote() {
			b, err := r.Reference(n, true)
			if err != nil {
				return err
			}
			v, err := reaches(r.Repository, b.Hash(), commit.Hash(), memo)
			if err != nil {
				return err
			}
			if v {
				branches = append(branches, n)
			}
		}
		return nil
	})

	for _, b := range branches {
		// If there are multiple branches, prefer the local branch over the remote branch.
		// If there are multiple local branches, simply return the first one we encounter.
		if b.IsBranch() {
			return b.Short(), nil
		}

		// If there are no local branches, return the first remote branch we encounter with the
		// remote name stripped.
		remotes, err := r.Remotes()
		if err != nil {
			return "", fmt.Errorf("failed to get remotes: %w", err)
		}
		for _, remote := range remotes {
			branch, match := strings.CutPrefix(b.Short(), remote.Config().Name+"/")
			if match {
				return branch, nil
			}
		}
	}
	return "", fmt.Errorf("no branches found")
}

// reaches returns true if commit, c, can be reached from commit, start. Results are memoized in memo.
func reaches(r *git.Repository, start, c plumbing.Hash, memo map[plumbing.Hash]bool) (bool, error) {
	if v, ok := memo[start]; ok {
		return v, nil
	}
	if start == c {
		memo[start] = true
		return true, nil
	}
	co, err := r.CommitObject(start)
	if err != nil {
		return false, err
	}
	for _, p := range co.ParentHashes {
		v, err := reaches(r, p, c, memo)
		if err != nil {
			return false, err
		}
		if v {
			memo[start] = true
			return true, nil
		}
	}
	memo[start] = false
	return false, nil
}

func getCurrentBranchFromRepository(repository *Git) (string, error) {
	branchRefs, err := repository.Branches()
	if err != nil {
		return "", err
	}

	headRef, err := repository.Head()
	if err != nil {
		return "", err
	}

	var currentBranchName string
	err = branchRefs.ForEach(func(branchRef *plumbing.Reference) error {
		if branchRef.Hash() == headRef.Hash() {
			currentBranchName = branchRef.Name().String()

			return nil
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return strings.TrimPrefix(currentBranchName, "refs/heads/"), nil
}

func getCurrentCommitFromRepository(repository *Git) (string, error) {
	headRef, err := repository.Head()
	if err != nil {
		return "", err
	}
	headSha := headRef.Hash().String()[:8]

	return headSha, nil
}

func getLatestTagFromRepository(repository *Git) (string, int, string, error) {
	// Check if HEAD has tag before iterating over all tags
	headRef, err := repository.Head()
	if err != nil {
		return "", 0, "", err
	}

	tag, count, latestHash, err := repository.Describe(headRef)
	if err != nil {
		return "", 0, "", err
	}

	return tag, count, latestHash, nil
}
