# gitlab-json-inheritance-go
A Golang implementation for resolving GitLabs Json inheritance syntax


Example

```go
package main

import (
	"fmt"
	"log"
	"yourmodule/configresolver"
)

func main() {
	// Setup GitLab fetcher (replace with your values)
	fetcher, err := configresolver.NewGitLabFetcher(
		"https://gitlab.com/api/v4",
		"your_access_token",
		"your_project_id",
	)
	if err != nil {
		log.Fatalf("Failed to create fetcher: %v", err)
	}

	configJSON := `{
		"extends": ["gitlab>html-validate/renovate-config"],
		"rules": {
			"maxLineLength": 100
		}
	}`

	resolved, err := configresolver.ResolveConfigStringWithFetcher(configJSON, fetcher)
	if err != nil {
		log.Fatalf("Failed to resolve config: %v", err)
	}

	fmt.Printf("Resolved config: %+v\n", resolved)
}

```