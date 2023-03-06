package main

import (
	"log"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

// func getBranchFromCIEnv() string {

func isCI() bool {
	// This is the simplest way to detect if we are in a CI environment it seems
	_, s := os.LookupEnv("CI")
	return s
}

// getRefsFromCI returns the commit, branch name, and tag name, if available, from the CI environment
func getRefsFromCI(r *Git) (string, string, string) {
	// Check if we are in a pull request ref
	rs, err := r.References()

	if err != nil {
		log.Fatal(err)
	}

	var commit, branch, tag string

	rs.ForEach(func(ref *plumbing.Reference) error {
		if strings.HasPrefix(ref.Name().String(), "refs/pull/") {
			// Probably in a pull request in Github
			// https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/checking-out-pull-requests-locally

			// Make absolutely sure it's in Gitlab
			if _, exists := os.LookupEnv("GITLAB_CI"); !exists {
				// Continue processing refs. Will probably result in no refs found, but this is more graceful
				return nil
			}
			commit, branch, tag = getRefsFromGithubCI()
		}
		if strings.HasPrefix(ref.Name().String(), "refs/pipelines/") {
			// Probably in a merge request pipeline in Gitlab
			// https://docs.gitlab.com/ee/ci/pipelines/merge_request_pipelines.html

			// Make absolutely sure we're in a Github Actions workflow
			if _, exists := os.LookupEnv("GITHUB_ACTION"); !exists {
				// Continue processing refs. Will probably result in no refs found, but this is more graceful
				return nil
			}
			commit, branch, tag = getRefsFromGitlabCI()
		}
		return nil
	})

	return commit, branch, tag
}

func getRefsFromGithubCI() (string, string, string) {

	// https://help.github.com/en/actions/reference/environment-variables#default-environment-variables

	refType, exists := os.LookupEnv("GITHUB_REF_TYPE")
	if !exists {
		// Something has gone wrong, or we aren't actually in a Github CI environment
		log.Fatalf("Failed to find GITHUB_REF_TYPE")
	}

	commit := os.Getenv("GITHUB_SHA")

	switch refType {
	case "branch":
		return commit, os.Getenv("GITHUB_REF_NAME"), ""
	case "tag":
		return commit, "", os.Getenv("GITHUB_REF_NAME")
	default:
		return commit, "", ""
	}
}

func getRefsFromGitlabCI() (string, string, string) {
	// https://docs.gitlab.com/ee/ci/variables/predefined_variables.html

	// CI_COMMIT_SHA should always be present.
	commit := os.Getenv("CI_COMMIT_SHA")

	// CI_COMMIT_TAG is present only in tag pipelines
	if tag := os.Getenv("CI_COMMIT_TAG"); tag != "" {
		return commit, "", tag
	}

	// CI_COMMIT_BRANCH is present only in branch pipelines, including default branch.
	// Not availabe in Merge Request pipelines or tag pipelines
	if branch := os.Getenv("CI_COMMIT_BRANCH"); branch != "" {
		return commit, branch, ""
	}

	// CI_MERGE_REQUEST_SOURCE_BRANCH_NAME is only available in Merge Request pipelines
	// and is the source branch of the merge request.
	if branch := os.Getenv("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME"); branch != "" {
		return commit, branch, ""
	}

	// CI_EXTERNAL_PULL_REQUEST_SOURCE_REPOSITORY is only available in External Pull Request pipelines
	if branch := os.Getenv("CI_EXTERNAL_PULL_REQUEST_SOURCE_REPOSITORY"); branch != "" {
		return commit, branch, ""
	}

	return commit, "", ""

}
