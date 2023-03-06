/*
Mostly taken from/inspired by https://github.com/edupo/semver-cli
*/

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// Git struct wrapps Repository class from go-git to add a tag map used to perform queries when describing. From https://github.com/edupo/semver-cli/blob/master/gitWrapper/git.go
type Git struct {
	TagsMap map[plumbing.Hash][]*plumbing.Reference
	*git.Repository
}

// plainOpen opens a git repository from the given path. It detects if the
// repository is bare or a normal one. If the path doesn't contain a valid
// repository ErrRepositoryNotExists is returned
func plainOpen(path string) (*Git, error) {
	opts := &git.PlainOpenOptions{
		DetectDotGit: true,
	}
	r, err := git.PlainOpenWithOptions(path, opts)
	return &Git{
		make(map[plumbing.Hash][]*plumbing.Reference),
		r,
	}, err
}

func (g *Git) getTagMap() error {
	tags, err := g.Tags()
	if err != nil {
		return err
	}

	err = tags.ForEach(func(t *plumbing.Reference) error {
		references := []*plumbing.Reference{t}
		var hash plumbing.Hash
		annTag, err := g.TagObject(t.Hash())
		switch err {
		case nil:
			// Annotated Tag object
			// This means the tags TARGET is actually what we want to use
			hash = annTag.Target
			if _, exists := g.TagsMap[hash]; exists {
				references = append(g.TagsMap[hash], t)
			}
		case plumbing.ErrObjectNotFound:
			// Not an annotated tag object
			// This means the tags hash is the hash of the actual commit
			hash = t.Hash()
			if _, exists := g.TagsMap[hash]; exists {
				references = append(g.TagsMap[hash], t)
			}
		default:
			// Some other error
			return err
		}
		g.TagsMap[hash] = references
		return nil
	})

	return err
}

// Describe returns the latest tag and number of commits from reference to that tag.
// Almost the same as "git describe --tags" other than the "g" prefix on the commit hash
// If the reference itself has a tag the count will be empty
func (g *Git) Describe(reference *plumbing.Reference) (string, int, error) {

	// Fetch the reference log
	cIter, err := g.Log(&git.LogOptions{
		From:  reference.Hash(),
		Order: git.LogOrderCommitterTime,
	})

	if err != nil {
		return "", 0, err
	}

	// Build the tag map
	err = g.getTagMap()
	if err != nil {
		return "", 0, err
	}

	// Search the tag
	var tag *plumbing.Reference
	var count int
	err = cIter.ForEach(func(c *object.Commit) error {
		if ts, ok := g.TagsMap[c.Hash]; ok {
			if len(ts) != 1 {
				tag = getHighestSemverRef(ts)
			} else {
				tag = ts[0]
			}
		}
		if tag != nil {
			return storer.ErrStop
		}
		count++
		return nil
	})

	if err != nil {
		return "", 0, err
	}

	if tag != nil {
		if count == 0 {
			return fmt.Sprint(tag.Name().Short()), 0, nil
		}
		return tag.Name().Short(), count, nil
	}
	return "", 0, nil
}

func getCurrentCommitFromRepository(repository *Git) (string, error) {
	headRef, err := repository.Head()
	if err != nil {
		return "", err
	}
	headSha := headRef.Hash().String()[:8]

	return headSha, nil
}

func getLatestTagFromRepository(repository *Git) (string, int, error) {
	// Check if HEAD has tag before iterating over all tags
	headRef, err := repository.Head()
	if err != nil {
		return "", 0, err
	}

	tag, count, err := repository.Describe(headRef)
	if err != nil {
		return "", 0, err
	}

	return tag, count, nil
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

func isDetachedHead(r *Git) bool {
	head, err := r.Head()
	if err != nil {
		return false
	}
	return head.Name() == plumbing.HEAD
}

func getCurrentBranchFromDetachedHead(r *Git) (string, error) {
	hr, err := r.Head()
	if err != nil {
		return "", err
	}

	memo := make(map[plumbing.Hash]bool)

	rs, err := r.References()

	if err != nil {
		return "", fmt.Errorf("no branch found in detached head at %q, err %w\n", hr.Hash().String()[:8], err)
	}

	var branches []plumbing.ReferenceName

	rs.ForEach(func(ref *plumbing.Reference) error {
		n := ref.Name()
		if n.IsBranch() || n.IsRemote() {
			b, err := r.Reference(n, true)
			if err != nil {
				return err
			}
			v, err := reaches(r.Repository, b.Hash(), hr.Hash(), memo)
			if err != nil {
				return err
			}
			if v {
				branches = append(branches, n)
			}
		}
		return nil
	})

	var branch string
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
			b, match := strings.CutPrefix(b.Short(), remote.Config().Name+"/")
			if match {
				branch = b
				break
			}
		}
	}
	return branch, nil
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

func getHighestSemverRef(references []*plumbing.Reference) *plumbing.Reference {
	if len(references) == 0 {
		return &plumbing.Reference{}
	}

	refVer := make(map[string]*plumbing.Reference)

	for _, ref := range references {
		refVer[ref.Name().Short()] = ref
	}

	vs := make([]*semver.Version, len(references))
	for i, r := range references {
		v, err := semver.NewVersion(r.Name().Short())
		if err != nil {
			// One of the strings is not semver formatted, ignore it
			continue
		}

		vs[i] = v
	}
	sort.Sort(sort.Reverse(semver.Collection(vs)))
	return refVer[vs[0].Original()]
}
