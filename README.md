# tv - tmux session manager

`tv` is a fast, compiled Go application for finding and managing `tmux` sessions for projects located in a specified directory (defaults to `~/Templates`). It uses a Breadth-First Search (BFS) algorithm for efficient project discovery and `fzf` for interactive selection.

## Features

*   Fast project discovery using a compiled Go binary.
*   Interactive project selection with `fzf`.
*   Automatic `tmux` session creation and management.
*   Support for custom `tmux` window layouts via a `.tmux-sessionizer` file in the project directory.
*   Simple mode (`-s`) to open a project in `lf` without `tmux`.
*   Customizable project directory via the `-t` flag.

## Installation

### Prerequisites

*   Go
*   `fzf`
*   `tmux`
*   `lf`

### Building

To build the `tv` binary:

```bash
make build
```

### Installing

To install the `tv` binary to `/usr/local/bin`:

```bash
sudo make install
```

### Uninstalling

To uninstall the `tv` binary:

```bash
sudo make uninstall
```

## Usage

```
Usage: tv [options]
Open a tmux session for a project.
Options:
  -m string
        Master Directory to search for projects (default "~/Templates")
  -p string
        path to project
  -s    Simple mode. no tmux, only one lf window
```
