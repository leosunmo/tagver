package main

import (
	"log"
	"os"
)

// func getBranchFromCIEnv() string {

func isCI() bool {
	// This is the simplest way to detect if we are in a CI environment it seems
	_, s := os.LookupEnv("CI")
	return s
}

// getRefsFromCI returns the commit, branch name, and tag name, if available, from the CI environment
func getRefsFromCI() (string, string, string) {
	var commit, branch, tag string
	// Check if we're in Gitlab CI
	if _, exists := os.LookupEnv("GITLAB_CI"); exists {
		commit, branch, tag = getRefsFromGitlabCI()
	}
	// Check if we're in a Github Actions workflow
	if _, exists := os.LookupEnv("GITHUB_ACTION"); exists {
		commit, branch, tag = getRefsFromGithubCI()
	}

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
