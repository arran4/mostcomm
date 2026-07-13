package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func handleSkillUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	force := fs.Bool("force", false, "Force update even if local modifications exist")
	all := fs.Bool("all", false, "Update all installed skills")
	scope := fs.String("scope", "user", "Installation scope (user or project)")
	agent := fs.String("agent", "mostcomm", "Target agent")

	fs.Parse(args)

	baseDir, err := GetSkillDir(*scope, *agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving skill directory: %v\n", err)
		os.Exit(1)
	}

	var skillsToUpdate []string
	if *all {
		entries, err := os.ReadDir(baseDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					skillsToUpdate = append(skillsToUpdate, entry.Name())
				}
			}
		}
	} else {
		remaining := fs.Args()
		if len(remaining) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: mostcomm skill update <skill-name> [--force] [--all]")
			os.Exit(1)
		}
		skillsToUpdate = append(skillsToUpdate, remaining[0])
	}

	for _, skillName := range skillsToUpdate {
		updateSingleSkill(baseDir, skillName, *force)
	}
}

func updateSingleSkill(baseDir, skillName string, force bool) {
	skillDir := filepath.Join(baseDir, skillName)
	metadata, err := ReadMetadata(skillDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Error reading metadata: %v\n", skillName, err)
		return
	}

	if metadata.Revision == "local" {
		fmt.Printf("[%s] Skill is locally sourced, skipping update.\n", skillName)
		return
	}

	parsed, err := parseSkillSource(metadata.Source, metadata.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Error parsing source: %v\n", skillName, err)
		return
	}

	latestRevision, needsUpdate, err := CheckUpdateRemote(parsed.Owner, parsed.Repo, metadata.Revision)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Error checking for updates: %v\n", skillName, err)
		return
	}

	if !needsUpdate {
		fmt.Printf("[%s] Already up-to-date (revision: %s).\n", skillName, metadata.Revision[:8])
		return
	}

	// Check for local modifications
	modified := false
	for relPath, expectedHash := range metadata.FileHashes {
		fullPath := filepath.Join(skillDir, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			modified = true
			break
		}
		actualHash, err := ComputeFileHash(fullPath)
		if err != nil || actualHash != expectedHash {
			modified = true
			break
		}
	}

	if modified && !force {
		fmt.Fprintf(os.Stderr, "[%s] installed skill has local modifications; rerun with --force to replace\n", skillName)
		return
	}

	fmt.Printf("[%s] Updating from %s to %s...\n", skillName, metadata.Revision[:8], latestRevision[:8])

	// Create a temporary directory for the update on the same filesystem
	tmpDir, err := os.MkdirTemp(filepath.Dir(skillDir), "mostcomm-skill-update-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Error creating temp dir: %v\n", skillName, err)
		return
	}
	defer os.RemoveAll(tmpDir)

	if metadata.Revision == "embedded" {
		err = extractEmbeddedSkill(tmpDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] Error extracting update: %v\n", skillName, err)
			return
		}
	} else {
		_, err = fetchAndExtractGitHub(parsed.Owner, parsed.Repo, parsed.SubPath, tmpDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] Error fetching update: %v\n", skillName, err)
			return
		}
	}

	// Atomically replace directory using os.Rename.
	// Move the old one to a backup path first, then rename the new one into place.
	backupDir := skillDir + ".backup"
	os.RemoveAll(backupDir) // ensure no old backup exists
	err = os.Rename(skillDir, backupDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Error backing up old skill: %v\n", skillName, err)
		return
	}

	err = os.Rename(tmpDir, skillDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Error applying update: %v\n", skillName, err)
		// Try to restore backup
		os.Rename(backupDir, skillDir)
		return
	}

	// Remove backup
	os.RemoveAll(backupDir)

	// Recompute hashes
	hashes := make(map[string]string)
	filepath.Walk(skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() != "metadata.json" {
			relPath, _ := filepath.Rel(skillDir, path)
			hash, _ := ComputeFileHash(path)
			hashes[relPath] = hash
		}
		return nil
	})

	metadata.Revision = latestRevision
	metadata.FileHashes = hashes
	metadata.InstalledAt = time.Now()

	WriteMetadata(skillDir, metadata)

	fmt.Printf("[%s] Successfully updated.\n", skillName)
}
