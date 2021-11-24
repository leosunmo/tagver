/*
Mostly taken from/inspired by https://github.com/edupo/semver-cli
*/

package main

import (
	"fmt"
	"sort"

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

// Describe returns the latest tag, number of commits from reference to that tag, and reference's hash.
// Almost the same as "git describe --tags" other than the "g" prefix on the commit hash
// If the reference itself has a tag the count and hash will be empty
func (g *Git) Describe(reference *plumbing.Reference) (string, int, string, error) {

	// Fetch the reference log
	cIter, err := g.Log(&git.LogOptions{
		From:  reference.Hash(),
		Order: git.LogOrderCommitterTime,
	})

	if err != nil {
		return "", 0, "", err
	}

	// Build the tag map
	err = g.getTagMap()
	if err != nil {
		return "", 0, "", err
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
		return "", 0, "", err
	}

	if tag != nil {
		if count == 0 {
			return fmt.Sprint(tag.Name().Short()), 0, "", nil
		}
		return tag.Name().Short(), count, reference.Hash().String()[0:8], nil
	}
	return "", 0, "", nil
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
