package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func handleSkillInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	scope := fs.String("scope", "user", "Installation scope (user or project)")
	agent := fs.String("agent", "mostcomm", "Target agent")

	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: mostcomm skill inspect <skill-name> [--scope user|project] [--agent name]")
		os.Exit(1)
	}

	skillName := remaining[0]

	baseDir, err := GetSkillDir(*scope, *agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving skill directory: %v\n", err)
		os.Exit(1)
	}

	skillDir := filepath.Join(baseDir, skillName)
	metadata, err := ReadMetadata(skillDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading metadata for skill '%s': %v\n", skillName, err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}
