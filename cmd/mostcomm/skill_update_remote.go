package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CheckUpdateRemote compares the local revision with the upstream GitHub revision.
// Returns (latestRevision, needsUpdate, error).
func CheckUpdateRemote(owner, repo, currentRevision string) (string, bool, error) {
	if currentRevision == "local" {
		return "local", false, nil // Local skills cannot be updated via GitHub API
	}

	commitURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/HEAD", owner, repo)
	req, err := http.NewRequest("GET", commitURL, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("failed to fetch latest commit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var commit githubCommit
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return "", false, fmt.Errorf("failed to decode commit info: %w", err)
	}

	latestRevision := commit.Sha
	needsUpdate := latestRevision != currentRevision

	return latestRevision, needsUpdate, nil
}
