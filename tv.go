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
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
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

func main() {
	var simpleMode bool
	var projectPathArg string
	var MasterDir string

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Open a tmux session for a project.\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.BoolVar(&simpleMode, "s", false, "Simple mode. no tmux, only one lf window")
	flag.StringVar(&projectPathArg, "p", "", "path to project")
	flag.StringVar(&MasterDir, "m", "~/Templates", "Master Directory to search for projects")
	flag.Parse()

	home, err := os.UserHomeDir()
	if err != nil {
		os.Exit(1)
	}

	if strings.HasPrefix(MasterDir, "~/") {
		MasterDir = filepath.Join(home, MasterDir[2:])
	}

	selectedPath := projectPathArg

	if selectedPath == "" {
		MasterPath := MasterDir
		MasterDepth := len(strings.Split(MasterPath, string(os.PathSeparator)))
		const maxDepth = 6

		var queue []string
		var projectDirs []string

		initialEntries, err := os.ReadDir(MasterPath)
		if err != nil {
			os.Exit(0)
		}

		for _, entry := range initialEntries {
			if entry.IsDir() {
				queue = append(queue, filepath.Join(MasterPath, entry.Name()))
			}
		}

		for len(queue) > 0 {
			dir := queue[0]
			queue = queue[1:]

			currentDepth := len(strings.Split(dir, string(os.PathSeparator))) - MasterDepth

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

		if len(projectDirs) == 0 {
			os.Exit(0)
		}

		fzfInput := strings.Join(projectDirs, "\n")
		cmd := exec.Command("fzf", "--preview", "ls -la {} | head -20", "--preview-window=right:50%")
		cmd.Stdin = strings.NewReader(fzfInput)
		cmd.Stderr = os.Stderr

		output, err := cmd.Output()
		if err != nil {
			os.Exit(1)
		}

		selectedPath = string(bytes.TrimSpace(output))
		if selectedPath == "" {
			os.Exit(0)
		}
	}

	if simpleMode {
		lfCmd := exec.Command("lf", selectedPath)
		lfCmd.Stdin = os.Stdin
		lfCmd.Stdout = os.Stdout
		lfCmd.Stderr = os.Stderr
		lfCmd.Run()
		os.Exit(0)
	}

	sessionName := strings.ReplaceAll(filepath.Base(selectedPath), ".", "_")

	if !hasSession(sessionName) {
		tmuxNewSessionCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-n", "editor", "-c", selectedPath, "nvim .")
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
			{"AI", "gemini -m gemini-2.5-flash"},
			{"Security", "$SHELL -c \"echo 'Use <semgrep ci>, <snyk test>, or <gemini>'; exec $SHELL\""},
		}

		for i, win := range windows {
			windowNum := i + 2
			args := []string{"new-window", "-t", fmt.Sprintf("=%s:%d", sessionName, windowNum), "-n", win.name, "-c", selectedPath}
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

		muxSelectWindowCmd := exec.Command("tmux", "select-window", "-t", fmt.Sprintf("=%s:1", sessionName))
		muxSelectWindowCmd.Run()

		hydrate(sessionName, selectedPath)
	}

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
