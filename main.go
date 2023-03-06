package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var getDefault bool
var path string

var (
	getTag           = flag.Bool("t", false, "Return the latest semver tag (annotated or lightweight Git tag) (default)")
	getBranch        = flag.Bool("b", false, "Return the current branch")
	getCommit        = flag.Bool("c", false, "Return the current commit")
	ignoreUncleanTag = flag.Bool("ignore-unclean-tag", false, "Return only tag name even if the latest tag doesn't point to HEAD (\"v1.0.4\" instead of \"v1.0.4-1-89c22b28\")")
)

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

	flag.Parse()

	if !*getTag && !*getBranch && !*getCommit {
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

	var commit string
	var branch string
	var tag string
	var count int

	// Check if we're in a CI environment
	if isCI() {
		commit, branch, tag = getRefsFromCI(r)
	} else {
		if isDetachedHead(r) {
			// Check if we're in a detached head state
			branch, err = getCurrentBranchFromDetachedHead(r)
			if err != nil {
				log.Fatalf("Failed to get current branch from %s, err %s\n", path, err.Error())
			}
		} else {
			branch, err = getCurrentBranchFromRepository(r)
			if err != nil {
				log.Fatalf("Failed to get current branch from %q, %s\n", path, err.Error())
			}
		}
		commit, err = getCurrentCommitFromRepository(r)
		if err != nil {
			log.Fatalf("Failed to get current commit from %s, err %s\n", path, err.Error())
		}

		tag, count, err = getLatestTagFromRepository(r)
		if err != nil {
			if errors.Is(err, plumbing.ErrObjectNotFound) {
				tag = ""
			} else {
				log.Fatalf("Failed to get latest tag from %s, err %s\n", path, err.Error())
			}
		}
	}

	var idents []string

	if *getTag || getDefault {
		if tag != "" {
			idents = append(idents, tag)
		}
	}

	if tag == "" && getDefault {
		*getBranch = true
		*getCommit = true
	}

	if *getBranch {
		if branch != "" {
			idents = append(idents, branch)
		}
	}

	if (*getTag || getDefault) && !*ignoreUncleanTag && count != 0 {
		idents = append(idents, strconv.Itoa(count))
		if !*getCommit {
			// Forcefully add commit even if it's not desired since we're
			// in an unclean state.
			idents = append(idents, commit)
		}
	}

	if *getCommit || getDefault {
		idents = append(idents, commit)
	}

	if len(idents) == 0 {
		log.Println("no version information found")
		return
	}
	fmt.Printf("%s\n", strings.Join(idents, "-"))

}
