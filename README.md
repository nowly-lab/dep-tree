# Dep Tree

[![Coverage Status](https://coveralls.io/repos/github/gabotechs/dep-tree/badge.svg?branch=main)](https://coveralls.io/github/gabotechs/dep-tree?branch=main)
![](https://img.shields.io/github/v/release/gabotechs/dep-tree?color=%e535abff)

Render your project's dependency tree in the terminal and/or validate it against your rules.

<p align="center">
    <img src="docs/demo.gif" alt="Dependency tree render">
</p>

## Install

Currently, only install through brew is supported:
```shell
brew install gabotechs/taps/dep-tree
```

## Usage

With dep-tree you can either render an interactive dependency tree in your terminal, or check
that your project's dependency graph matches some user defined rules.

### Render

Choose the file that will act as the root of the dependency tree and run:

```shell
dep-tree render my-file.js
```

### Dependency check

Create a configuration file `.dep-tree-yml` with some rules in it:

```yml
entrypoints:
  - src/index.ts
white_list:
  "src/utils/**/*.ts":
    - "src/utils/**/*.ts"  # The files in src/utils can only depend on other utils
black_list:
  "src/ports/**/*.ts":
    - "**"  # A port cannot have any dependency
```

and check that your project matches that rules:

```shell
dep-tree check
```

## Supported languages

- JavaScript/TypeScript
- Python (coming soon...)
- Rust (coming soon...)
- Golang (coming soon...)
