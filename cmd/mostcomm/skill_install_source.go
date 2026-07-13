package main

import (
	"fmt"
	"path/filepath"
	"os"
	"strings"
)

type ParsedSource struct {
	IsLocal      bool
	LocalPath    string
	Owner        string
	Repo         string
	SubPath      string
	SkillName    string
	GivenName    string
}

func parseSkillSource(source string, optionalName string) (*ParsedSource, error) {
	parsed := &ParsedSource{}

	// Check if local path
	if strings.HasPrefix(source, ".") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "~") {
		parsed.IsLocal = true
		parsed.LocalPath = source
		if strings.HasPrefix(source, "~") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				parsed.LocalPath = filepath.Join(homeDir, strings.TrimPrefix(source, "~/"))
			}
		}
		// Verify directory exists
		info, err := os.Stat(parsed.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("invalid local source: %w", err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("local source %s is not a directory", source)
		}

		if optionalName != "" {
			parsed.SkillName = optionalName
			parsed.GivenName = optionalName
		} else {
			// Extract from path
			parts := strings.Split(strings.TrimRight(source, "/"), "/")
			parsed.SkillName = parts[len(parts)-1]
		}
		return parsed, nil
	}

	// GitHub format: owner/repo [path/to/skill]
	parts := strings.SplitN(source, "/", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid remote source format. Use owner/repo[/path]")
	}

	parsed.Owner = parts[0]
	parsed.Repo = parts[1]

	if len(parts) == 3 {
		parsed.SubPath = parts[2]
	}

	if optionalName != "" {
		parsed.GivenName = optionalName
		parsed.SkillName = optionalName
	} else if parsed.SubPath != "" {
		pathParts := strings.Split(strings.TrimRight(parsed.SubPath, "/"), "/")
		parsed.SkillName = pathParts[len(pathParts)-1]
	} else {
		parsed.SkillName = parsed.Repo // fallback to repo name
	}

	return parsed, nil
}
