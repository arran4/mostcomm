package main

import (
	"mostcomm"
	"os"
)

func extractEmbeddedSkill(destDir string) error {
	data, err := mostcomm.EmbeddedSkills.ReadFile("skills/mostcomm/SKILL.md")
	if err != nil {
		return err
	}

	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(destDir+"/SKILL.md", data, 0644)
}
