package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// Basic example of how to list tags.
func main() {
	var getTag, getBranch, getCommit bool
	var path string

	flag.BoolVar(&getTag, "t", false, "Return the latest semver tag (annotated or lightweight Git tag) (default)")
	flag.BoolVar(&getBranch, "b", false, "Return the current branch")
	flag.BoolVar(&getCommit, "c", false, "Return the current commit")
	flag.StringVar(&path, "path", "./", "Path of Git repository (Default current directory)")

	flag.Parse()

	if !getTag && !getBranch && !getCommit {
		getTag = true
	}

	// We instance a new repository targeting the given path (the .git folder)
	r, err := git.PlainOpen(path)
	if err != nil {
		log.Fatalln("Failed to open path")
	}

	var commit, branch, tag string

	if getCommit {
		var err error
		commit, err = getCurrentCommitFromRepository(r)
		if err != nil {
			log.Fatalf("Failed to get current commit from %s, err %s\n", path, err.Error())
		}
	}

	if getBranch {
		var err error
		branch, err = getCurrentBranchFromRepository(r)
		if err != nil {
			log.Fatalf("Failed to get current branch from %s, err %s\n", path, err.Error())
		}
	}

	if getTag {
		var err error
		tag, err = getLatestTagFromRepository(r)
		if err != nil {
			log.Fatalf("Failed to get latest tag from %s, err %s\n", path, err.Error())
		}
	}
	pt := []string{tag, branch, commit}
	var imageTag []string
	for _, s := range pt {
		if s != "" {
			imageTag = append(imageTag, s)
		}
	}
	fmt.Printf("%s\n", strings.Join(imageTag, "-"))
}

func getCurrentBranchFromRepository(repository *git.Repository) (string, error) {
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

func getCurrentCommitFromRepository(repository *git.Repository) (string, error) {
	headRef, err := repository.Head()
	if err != nil {
		return "", err
	}
	headSha := headRef.Hash().String()[:8]

	return headSha, nil
}

func getLatestTagFromRepository(repository *git.Repository) (string, error) {
	tagRefs, err := repository.Tags()
	if err != nil {
		return "", err
	}

	var tagCandidate *object.Commit
	err = tagRefs.ForEach(func(tagRef *plumbing.Reference) error {
		revision := plumbing.Revision(tagRef.Name().String())
		tagCommitHash, err := repository.ResolveRevision(revision)
		if err != nil {
			return err
		}

		commit, err := repository.CommitObject(*tagCommitHash)
		if err != nil {
			return err
		}

		if tagCandidate == nil {
			tagCandidate = commit
		}

		if commit.Committer.When.After(tagCandidate.Committer.When) {
			tagCandidate = commit
		}

		return nil
	})
	if err != nil {
		return "", err
	}
	tagRefs, err = repository.Tags()
	if err != nil {
		return "", err
	}
	var potentials []string
	// Special case if there's more than one tag (annotated or lightweight) against a single commit
	// since the Commiter.When will be the same on both tags, try to find if one has a higher number
	err = tagRefs.ForEach(func(tagRef *plumbing.Reference) error {
		revision := plumbing.Revision(tagRef.Name().String())
		tagCommitHash, err := repository.ResolveRevision(revision)
		if err != nil {
			return err
		}

		commit, err := repository.CommitObject(*tagCommitHash)
		if err != nil {
			return err
		}

		if commit.Hash.String() == tagCandidate.Hash.String() {
			potentials = append(potentials, tagRef.Name().Short())
		}

		return nil
	})

	return getHighestSemver(potentials), nil
	//return strings.TrimPrefix(latestTagName, "refs/tags/"), nil
}

func getHighestSemver(semvers []string) string {
	if len(semvers) == 0 {
		return ""
	}
	vs := make([]*semver.Version, len(semvers))
	for i, r := range semvers {
		v, err := semver.NewVersion(r)
		if err != nil {
			// One of the strings is not semver formatted, ignore it
			continue
		}

		vs[i] = v
	}
	sort.Sort(sort.Reverse(semver.Collection(vs)))
	return vs[0].Original()
}

func checkArgs(arg ...string) {
	if len(os.Args) < len(arg)+1 {
		log.Printf("Usage: %s %s\n", os.Args[0], strings.Join(arg, " "))
		os.Exit(1)
	}
}
