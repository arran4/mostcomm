package main

import (
	"fmt"
	"os"
)

func handleSkillCommand(args []string) {
	if len(args) == 0 {
		printSkillUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "install":
		handleSkillInstall(args[1:])
	case "update":
		handleSkillUpdate(args[1:])
	case "remove":
		handleSkillRemove(args[1:])
	case "list":
		handleSkillList(args[1:])
	case "inspect":
		handleSkillInspect(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown skill command: %s\n", args[0])
		printSkillUsage()
		os.Exit(1)
	}
}

func printSkillUsage() {
	fmt.Fprintln(os.Stderr, "Usage: mostcomm skill <command> [options]")
	fmt.Fprintln(os.Stderr, "\nCommands:")
	fmt.Fprintln(os.Stderr, "  install  Install a new skill")
	fmt.Fprintln(os.Stderr, "  update   Update an installed skill")
	fmt.Fprintln(os.Stderr, "  remove   Remove an installed skill")
	fmt.Fprintln(os.Stderr, "  list     List installed skills")
	fmt.Fprintln(os.Stderr, "  inspect  Inspect an installed skill")
}
