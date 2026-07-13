package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"flag"
)

func handleSkillInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	scope := fs.String("scope", "user", "Installation scope (user or project)")
	agent := fs.String("agent", "mostcomm", "Target agent")

	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: mostcomm skill install <source> [skill-name] [--scope user|project] [--agent name]")
		os.Exit(1)
	}

	sourceStr := remaining[0]
	optionalName := ""
	if len(remaining) > 1 {
		optionalName = remaining[1]
	}

	parsed, err := parseSkillSource(sourceStr, optionalName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing source: %v\n", err)
		os.Exit(1)
	}

	baseDir, err := GetSkillDir(*scope, *agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving skill directory: %v\n", err)
		os.Exit(1)
	}

	destDir := filepath.Join(baseDir, parsed.SkillName)

	if _, err := os.Stat(destDir); err == nil {
		fmt.Fprintf(os.Stderr, "Skill '%s' is already installed at %s\n", parsed.SkillName, destDir)
		os.Exit(1)
	}

	// Create a temporary directory for extraction
	tmpDir, err := os.MkdirTemp("", "mostcomm-skill-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	var revision string
	if parsed.IsLocal {
		err = copyLocalSkill(parsed.LocalPath, tmpDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error copying local skill: %v\n", err)
			os.Exit(1)
		}
		revision = "local"
	} else {
		revision, err = fetchAndExtractGitHub(parsed.Owner, parsed.Repo, parsed.SubPath, tmpDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching from GitHub: %v\n", err)
			os.Exit(1)
		}
	}

	// Check for SKILL.md
	if _, err := os.Stat(filepath.Join(tmpDir, "SKILL.md")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Invalid skill: missing SKILL.md in %s\n", sourceStr)
		os.Exit(1)
	}

	// Move tmpDir to destDir
	err = os.MkdirAll(filepath.Dir(destDir), 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating destination directory: %v\n", err)
		os.Exit(1)
	}

	err = os.Rename(tmpDir, destDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing skill: %v\n", err)
		os.Exit(1)
	}

	// Hash files and write metadata
	hashes := make(map[string]string)
	err = filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() != "metadata.json" {
			relPath, err := filepath.Rel(destDir, path)
			if err != nil {
				return err
			}
			hash, err := ComputeFileHash(path)
			if err != nil {
				return err
			}
			hashes[relPath] = hash
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error hashing files: %v\n", err)
		os.Exit(1)
	}

	metadata := &SkillMetadata{
		Name:        parsed.SkillName,
		Source:      sourceStr,
		Path:        destDir,
		Revision:    revision,
		InstalledAt: time.Now(),
		Agent:       *agent,
		Scope:       *scope,
		FileHashes:  hashes,
	}

	err = WriteMetadata(destDir, metadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing metadata: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully installed skill '%s' to %s\n", parsed.SkillName, destDir)
}

func copyLocalSkill(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_RDWR, info.Mode())
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

// GitHub API response for commits
type githubCommit struct {
	Sha string `json:"sha"`
}

func fetchAndExtractGitHub(owner, repo, subPath, destDir string) (string, error) {
	// 1. Get latest commit SHA for revision tracking
	commitURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/HEAD", owner, repo)
	req, err := http.NewRequest("GET", commitURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch commit info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var commit githubCommit
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return "", fmt.Errorf("failed to decode commit info: %w", err)
	}
	revision := commit.Sha

	// 2. Fetch tarball
	tarballURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball/%s", owner, repo, revision)
	reqTar, err := http.NewRequest("GET", tarballURL, nil)
	if err != nil {
		return "", err
	}

	respTar, err := client.Do(reqTar)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tarball: %w", err)
	}
	defer respTar.Body.Close()

	if respTar.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch tarball, status: %d", respTar.StatusCode)
	}

	// 3. Extract tarball
	gz, err := gzip.NewReader(respTar.Body)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	var rootDir string
	foundFiles := false

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Skip pax_global_header
		if hdr.Name == "pax_global_header" {
			continue
		}

		// The first component is the dynamic root directory (e.g. owner-repo-sha)
		parts := strings.SplitN(hdr.Name, "/", 2)
		if len(parts) == 0 {
			continue
		}

		if rootDir == "" {
			rootDir = parts[0]
		}

		if len(parts) < 2 || parts[1] == "" {
			continue // just the root dir itself
		}

		relPath := parts[1]

		// If a subPath was specified, only extract files within it
		if subPath != "" {
			if !strings.HasPrefix(relPath, subPath+"/") && relPath != subPath {
				continue
			}
			// Strip subPath from relPath
			if relPath == subPath {
				continue // skip the dir itself
			}
			relPath = strings.TrimPrefix(relPath, subPath+"/")
		}

		targetPath := filepath.Join(destDir, relPath)

		// Security: prevent path traversal
		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return "", fmt.Errorf("invalid file path in tarball: %s", hdr.Name)
		}

		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return "", err
			}
		} else if hdr.Typeflag == tar.TypeReg {
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return "", err
			}
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return "", err
			}
			f.Close()
			foundFiles = true
		}
	}

	if !foundFiles {
		return "", fmt.Errorf("no files found to install. Check if the subpath '%s' exists", subPath)
	}

	return revision, nil
}
