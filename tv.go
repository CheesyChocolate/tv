package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func hasSession(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", "="+sessionName)
	return cmd.Run() == nil
}

func hydrate(sessionName, projectPath string) {
	sessionizerPath := filepath.Join(projectPath, ".tmux-sessionizer")
	if _, err := os.Stat(sessionizerPath); err == nil {
		// Source the .tmux-sessionizer file
		// This is tricky to do directly in Go as it modifies the shell environment.
		cmd := exec.Command("tmux", "send-keys", "-t", "="+sessionName, fmt.Sprintf("source %s", sessionizerPath), "C-m")
		cmd.Run()
	}
}

func findProjects(masterPath string) ([]string, error) {
	masterDepth := len(strings.Split(masterPath, string(os.PathSeparator)))
	const maxDepth = 6

	var queue []string
	var projectDirs []string

	initialEntries, err := os.ReadDir(masterPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range initialEntries {
		if entry.IsDir() {
			queue = append(queue, filepath.Join(masterPath, entry.Name()))
		}
	}

	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]

		currentDepth := len(strings.Split(dir, string(os.PathSeparator))) - masterDepth

		var hasFiles bool
		var subdirs []string

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				hasFiles = true
				break
			}
			if currentDepth < maxDepth {
				subdirs = append(subdirs, filepath.Join(dir, entry.Name()))
			}
		}

		if hasFiles || currentDepth >= maxDepth {
			projectDirs = append(projectDirs, dir)
		} else {
			queue = append(queue, subdirs...)
		}
	}
	return projectDirs, nil
}

func selectProject(projects []string) (string, error) {
	fzfInput := strings.Join(projects, "\n")
	cmd := exec.Command("fzf", "--preview", "ls -la {} | head -20", "--preview-window=right:50%")
	cmd.Stdin = strings.NewReader(fzfInput)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
}

func createSession(sessionName, projectPath string) {
	if hasSession(sessionName) {
		return
	}

	tmuxNewSessionCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-n", "editor", "-c", projectPath, "nvim . ")
	tmuxNewSessionCmd.Stdin = os.Stdin
	tmuxNewSessionCmd.Stdout = os.Stdout
	tmuxNewSessionCmd.Stderr = os.Stderr
	if err := tmuxNewSessionCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating new tmux session: %v\n", err)
		os.Exit(1)
	}

	windows := []struct {
		name    string
		command string
	}{
		{"shell", ""},
		{"git", "nvim -c MagitOnly"},
		{"lf", "lf"},
		{"AI", "gemini"},
		{"Security", "$SHELL -c \"echo 'Use <semgrep ci>, <snyk test>, or <gemini>'; exec $SHELL\""},
	}

	for i, win := range windows {
		windowNum := i + 2
		args := []string{"new-window", "-t", fmt.Sprintf("=%s:%d", sessionName, windowNum), "-n", win.name, "-c", projectPath}
		if win.command != "" {
			args = append(args, win.command)
		}
		tmuxNewWindowCmd := exec.Command("tmux", args...)
		tmuxNewWindowCmd.Stdin = os.Stdin
		tmuxNewWindowCmd.Stdout = os.Stdout
		tmuxNewWindowCmd.Stderr = os.Stderr
		if err := tmuxNewWindowCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating tmux window %s: %v\n", win.name, err)
		}
	}

	tmuxSelectWindowCmd := exec.Command("tmux", "select-window", "-t", fmt.Sprintf("=%s:1", sessionName))
	tmuxSelectWindowCmd.Run()

	hydrate(sessionName, projectPath)
}

func attachSession(sessionName string) {
	attachCmd := exec.Command("tmux", "attach-session", "-t", "="+sessionName)
	if os.Getenv("TMUX") != "" {
		attachCmd = exec.Command("tmux", "switch-client", "-t", "="+sessionName)
	}

	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr

	if err := attachCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error attaching to tmux session: %v\n", err)
		os.Exit(1)
	}
}

func resolveMasterDir(masterDir string) string {
	if strings.HasPrefix(masterDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			os.Exit(1)
		}
		return filepath.Join(home, masterDir[2:])
	}
	return masterDir
}

func runSimpleMode(path string) {
	lfCmd := exec.Command("lf", path)
	lfCmd.Stdin = os.Stdin
	lfCmd.Stdout = os.Stdout
	lfCmd.Stderr = os.Stderr
	lfCmd.Run()
	os.Exit(0)
}

func fixSessionName(path string) string {
	return strings.ReplaceAll(filepath.Base(path), ".", "_")
}

func main() {
	var simpleMode bool
	var projectPathArg string
	var masterDir string

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Open a tmux session for a project.\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.BoolVar(&simpleMode, "s", false, "Simple mode. no tmux, only one lf window")
	flag.StringVar(&projectPathArg, "p", "", "path to project")
	flag.StringVar(&masterDir, "m", "~/Templates", "Master Directory to search for projects")
	flag.Parse()

	masterDir = resolveMasterDir(masterDir)
	selectedPath := projectPathArg

	if selectedPath == "" {
		projects, err := findProjects(masterDir)
		if err != nil || len(projects) == 0 {
			os.Exit(0)
		}

		selected, err := selectProject(projects)
		if err != nil || selected == "" {
			os.Exit(0)
		}
		selectedPath = selected
	}

	if simpleMode {
		runSimpleMode(selectedPath)
	}

	if absPath, err := filepath.Abs(selectedPath); err == nil {
		selectedPath = absPath
	}

	sessionName := fixSessionName(selectedPath)

	createSession(sessionName, selectedPath)
	attachSession(sessionName)
}
