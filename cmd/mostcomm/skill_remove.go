package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func handleSkillRemove(args []string) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	scope := fs.String("scope", "user", "Installation scope (user or project)")
	agent := fs.String("agent", "mostcomm", "Target agent")

	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: mostcomm skill remove <skill-name> [--scope user|project] [--agent name]")
		os.Exit(1)
	}

	skillName := remaining[0]

	baseDir, err := GetSkillDir(*scope, *agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving skill directory: %v\n", err)
		os.Exit(1)
	}

	skillDir := filepath.Join(baseDir, skillName)

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Skill '%s' is not installed (scope: %s, agent: %s).\n", skillName, *scope, *agent)
		os.Exit(1)
	}

	err = os.RemoveAll(skillDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error removing skill '%s': %v\n", skillName, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully removed skill '%s' (scope: %s, agent: %s).\n", skillName, *scope, *agent)
}
