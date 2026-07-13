package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// SkillMetadata stores provenance and installation metadata for a skill.
type SkillMetadata struct {
	Name        string            `json:"name"`
	Source      string            `json:"source"`
	Path        string            `json:"path"`
	Revision    string            `json:"revision"`
	InstalledAt time.Time         `json:"installed_at"`
	Agent       string            `json:"agent"`
	Scope       string            `json:"scope"`
	FileHashes  map[string]string `json:"file_hashes"`
}

// GetSkillDir resolves the agent-specific installation directory.
func GetSkillDir(scope string, agent string) (string, error) {
	var baseDir string
	if scope == "user" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = homeDir
	} else if scope == "project" {
		baseDir = "."
	} else {
		return "", fmt.Errorf("invalid scope: %s (must be 'user' or 'project')", scope)
	}

	switch agent {
	case "copilot":
		return filepath.Join(baseDir, ".github", "copilot", "skills"), nil
	case "cursor":
		return filepath.Join(baseDir, ".cursor", "skills"), nil
	case "codex":
		return filepath.Join(baseDir, ".codex", "skills"), nil
	default:
		// Default agent skills convention
		return filepath.Join(baseDir, ".agents", "skills"), nil
	}
}

// ComputeFileHash computes the SHA256 hash of a file.
func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// WriteMetadata writes the skill metadata to metadata.json in the skill directory.
func WriteMetadata(skillPath string, metadata *SkillMetadata) error {
	metadataPath := filepath.Join(skillPath, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metadataPath, data, 0644)
}

// ReadMetadata reads the skill metadata from metadata.json in the skill directory.
func ReadMetadata(skillPath string) (*SkillMetadata, error) {
	metadataPath := filepath.Join(skillPath, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}

	var metadata SkillMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}
