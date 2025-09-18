package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		os.Exit(1)
	}

	templatesPath := filepath.Join(home, "Templates")
	templatesDepth := len(strings.Split(templatesPath, string(os.PathSeparator)))
	const maxDepth = 6

	var queue []string

	initialEntries, err := os.ReadDir(templatesPath)
	if err != nil {
		os.Exit(1)
	}

	for _, entry := range initialEntries {
		if entry.IsDir() {
			queue = append(queue, filepath.Join(templatesPath, entry.Name()))
		}
	}

	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:] // Dequeue

		currentDepth := len(strings.Split(dir, string(os.PathSeparator))) - templatesDepth

		var hasFiles bool
		var subdirs []string

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // Skip directories we can't read
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				hasFiles = true
				break // Found a file, no need to check further
			}
			// Only add the subdir path if we might need it later
			if currentDepth < maxDepth {
				subdirs = append(subdirs, filepath.Join(dir, entry.Name()))
			}
		}

		if hasFiles || currentDepth >= maxDepth {
			fmt.Println(dir)
		} else {
			queue = append(queue, subdirs...) // Enqueue subdirectories
		}
	}
}
