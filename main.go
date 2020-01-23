package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
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
		branch, err = getCurrentBranchFromRepository(r)
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
