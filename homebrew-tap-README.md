# Homebrew Tap for nowly-tree

This is the official Homebrew tap for `nowly-tree`, Nowly Lab's distribution of dep-tree.

## Installation

```bash
brew tap nowly-lab/dep-tree
brew install nowly-tree
```

## Usage

```bash
# Show version
nowly-tree --version

# Analyze dependencies
nowly-tree src/index.ts

# Generate dependency tree
nowly-tree tree package/main.py --unwrap-exports

# Check dependencies against rules
nowly-tree check
```

## Updating

```bash
brew update
brew upgrade nowly-tree
```

## About

`nowly-tree` is a tool for visualizing and analyzing project dependencies. It supports multiple programming languages including:

- JavaScript/TypeScript
- Python
- Rust
- Go

For more information, visit the [main repository](https://github.com/nowly-lab/dep-tree).

## Formula

The Homebrew formula is automatically updated by GoReleaser when new releases are published.