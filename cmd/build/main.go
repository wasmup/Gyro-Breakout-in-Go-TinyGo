// Command build creates a TinyGo target with the project-local vector table.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	tinygo := os.Getenv("TINYGO")
	if tinygo == "" {
		tinygo = "tinygo"
	}
	cmd := exec.Command(tinygo, "env", "TINYGOROOT")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("find TinyGo root: %w", err)
	}
	tinygoRoot := strings.TrimSpace(string(output))
	data, err := os.ReadFile(filepath.Join(root, "targets", "stm32f429.json"))
	if err != nil {
		return err
	}
	var target map[string]any
	if err := json.Unmarshal(data, &target); err != nil {
		return err
	}
	vector, err := filepath.Rel(tinygoRoot, filepath.Join(root, "targets", "stm32f429-vectors.S"))
	if err != nil {
		return err
	}
	target["extra-files"] = []string{vector}
	data, err = json.Marshal(target)
	if err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "f429-target-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	config := filepath.Join(temp, "target.json")
	if err := os.WriteFile(config, data, 0600); err != nil {
		return err
	}
	cmd = exec.Command(tinygo, "build", "-scheduler=none", "-target", config, "-o", filepath.Join(root, "build", "gyro-breakout-go.elf"), "./cmd/firmware")
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("TinyGo build: %w", err)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
