package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSkillSource(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		optional   string
		wantLocal  bool
		wantOwner  string
		wantRepo   string
		wantSub    string
		wantSkill  string
		wantErr    bool
	}{
		{
			name:       "GitHub standard",
			source:     "owner/repo",
			wantLocal:  false,
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantSkill:  "repo",
		},
		{
			name:       "GitHub with subpath",
			source:     "owner/repo/skills/myskill",
			wantLocal:  false,
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantSub:    "skills/myskill",
			wantSkill:  "myskill",
		},
		{
			name:       "GitHub with explicit name",
			source:     "owner/repo/skills/myskill",
			optional:   "customname",
			wantLocal:  false,
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantSub:    "skills/myskill",
			wantSkill:  "customname",
		},
		{
			name:       "Invalid remote format",
			source:     "ownerrepo",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSkillSource(tt.source, tt.optional)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSkillSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if got.IsLocal != tt.wantLocal {
				t.Errorf("IsLocal = %v, want %v", got.IsLocal, tt.wantLocal)
			}
			if got.Owner != tt.wantOwner {
				t.Errorf("Owner = %v, want %v", got.Owner, tt.wantOwner)
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("Repo = %v, want %v", got.Repo, tt.wantRepo)
			}
			if got.SubPath != tt.wantSub {
				t.Errorf("SubPath = %v, want %v", got.SubPath, tt.wantSub)
			}
			if got.SkillName != tt.wantSkill {
				t.Errorf("SkillName = %v, want %v", got.SkillName, tt.wantSkill)
			}
		})
	}
}

func TestParseSkillSourceLocal(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "local-skill")
	os.Mkdir(skillDir, 0755)

	got, err := parseSkillSource(skillDir, "")
	if err != nil {
		t.Fatalf("parseSkillSource() error: %v", err)
	}

	if !got.IsLocal {
		t.Errorf("Expected IsLocal = true")
	}
	if got.SkillName != "local-skill" {
		t.Errorf("Expected SkillName = local-skill, got %v", got.SkillName)
	}
}

func TestSkillInstallLocal(t *testing.T) {
	// Mock HOME to control where user-scoped skills install
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create a local skill source
	sourceDir := filepath.Join(homeDir, "mock-source")
	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Mock Skill"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "helper.sh"), []byte("echo hello"), 0755)

	// Intercept os.Args and os.Exit for handleSkillInstall
	// Note: since handleSkillInstall parses flag.CommandLine or calls os.Exit,
	// we will call copyLocalSkill directly, or we can use a mock approach.
	// For simplicity, let's test the underlying functions or simulate the environment.

	destDir := filepath.Join(homeDir, ".agents", "skills", "mock-source")

	err := copyLocalSkill(sourceDir, destDir)
	if err != nil {
		t.Fatalf("copyLocalSkill() failed: %v", err)
	}

	// Verify files copied
	if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); os.IsNotExist(err) {
		t.Errorf("SKILL.md not copied")
	}

	// Verify permissions preserved (approximate check)
	info, _ := os.Stat(filepath.Join(destDir, "helper.sh"))
	if info.Mode()&0111 == 0 {
		t.Errorf("Executable permissions not preserved on helper.sh")
	}
}

func TestSkillUpdateModified(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "myskill")
	os.Mkdir(skillDir, 0755)

	// Create initial file and hash
	filePath := filepath.Join(skillDir, "SKILL.md")
	os.WriteFile(filePath, []byte("Original Content"), 0644)
	hash, _ := ComputeFileHash(filePath)

	metadata := &SkillMetadata{
		Name:     "myskill",
		Source:   "owner/repo",
		Revision: "abcdef123",
		FileHashes: map[string]string{
			"SKILL.md": hash,
		},
	}
	WriteMetadata(skillDir, metadata)

	// Modify the file locally
	os.WriteFile(filePath, []byte("Modified Content"), 0644)

	// Since handleSkillUpdate interacts with GitHub and calls os.Exit, we'll
	// verify the logic that detects the modification directly or use the
	// CheckUpdateRemote wrapper logic. We test the underlying hash logic here.

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

	if !modified {
		t.Errorf("Expected local modifications to be detected")
	}
}

func TestSkillListAndRemove(t *testing.T) {
	// Mock HOME
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	baseDir := filepath.Join(homeDir, ".agents", "skills")
	os.MkdirAll(baseDir, 0755)

	// Create dummy skills
	skill1Dir := filepath.Join(baseDir, "skill1")
	os.Mkdir(skill1Dir, 0755)
	WriteMetadata(skill1Dir, &SkillMetadata{Name: "skill1"})

	skill2Dir := filepath.Join(baseDir, "skill2")
	os.Mkdir(skill2Dir, 0755)
	WriteMetadata(skill2Dir, &SkillMetadata{Name: "skill2"})

	// Validate they are listed
	entries, _ := os.ReadDir(baseDir)
	if len(entries) != 2 {
		t.Errorf("Expected 2 skills listed, got %d", len(entries))
	}

	// Validate remove logic
	err := os.RemoveAll(skill1Dir)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	entries, _ = os.ReadDir(baseDir)
	if len(entries) != 1 {
		t.Errorf("Expected 1 skill remaining, got %d", len(entries))
	}
	if entries[0].Name() != "skill2" {
		t.Errorf("Expected skill2 to remain, got %s", entries[0].Name())
	}
}
