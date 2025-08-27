package main

import (
	"flag"
		"flag"
		"fmt"
		"os"
		"github.com/frjcomp/gitlab-json-inheritance-go/configresolver"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to the JSON config file")
	gitlabURL := flag.String("gitlab-url", "https://gitlab.com/api/v4", "GitLab API base URL")
	gitlabToken := flag.String("gitlab-token", "", "GitLab access token")
	project := flag.String("project", "group/proj", "GitLab project path")
	branch := flag.String("branch", "main", "GitLab branch")
	flag.Parse()

	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read config file: %v\n", err)
		os.Exit(1)
	}

	fetcher, err := configresolver.NewGitLabFetcher(*gitlabURL, *gitlabToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create GitLab fetcher: %v\n", err)
		os.Exit(1)
	}

	resolved, err := configresolver.ResolveConfigStringWithFetcher(string(data), fetcher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Resolved config: %+v\n", resolved)
}
