package configresolver

import (
	"encoding/json"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Fetcher interface allows swapping between real GitLab and mocks
type Fetcher interface {
	Fetch(branch, projectPath string) (map[string]interface{}, error)
}

// GitLabFetcher fetches configs from GitLab
type GitLabFetcher struct {
	Client   *gitlab.Client
	FilePath string
}

func NewGitLabFetcher(baseURL, token string) (*GitLabFetcher, error) {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &GitLabFetcher{
		Client:   client,
		FilePath: ".gitlab/renovate.json",
	}, nil
}

// Fetch retrieves a config JSON from a project and branch
func (f *GitLabFetcher) Fetch(ref, project string) (map[string]interface{}, error) {
	// Use the official client-go method to fetch the raw file
	raw, _, err := f.Client.RepositoryFiles.GetRawFile(project, f.FilePath, &gitlab.GetRawFileOptions{Ref: &ref})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s@%s: %v", project, ref, err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s@%s: %v", project, ref, err)
	}
	return result, nil
}
