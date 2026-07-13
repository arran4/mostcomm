package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
)

func handleSkillList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	scope := fs.String("scope", "user", "Installation scope (user or project)")
	agent := fs.String("agent", "mostcomm", "Target agent")
	outputJSON := fs.Bool("json", false, "Output in JSON format")

	fs.Parse(args)

	baseDir, err := GetSkillDir(*scope, *agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving skill directory: %v\n", err)
		os.Exit(1)
	}

	var skills []*SkillMetadata

	entries, err := os.ReadDir(baseDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				skillDir := filepath.Join(baseDir, entry.Name())
				metadata, err := ReadMetadata(skillDir)
				if err == nil {
					skills = append(skills, metadata)
				}
			}
		}
	}

	if *outputJSON {
		data, err := json.MarshalIndent(skills, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	if len(skills) == 0 {
		fmt.Println("No skills installed.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tREVISION\tSCOPE")
	for _, skill := range skills {
		rev := skill.Revision
		if len(rev) > 8 && rev != "local" {
			rev = rev[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", skill.Name, skill.Source, rev, skill.Scope)
	}
	w.Flush()
}
