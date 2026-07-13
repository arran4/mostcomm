# Mostcomm Agent Skill

## Overview
`mostcomm` is a CLI tool that scans directories of text files to detect duplicated lines and blocks of content. It is extremely useful for finding copy-pasted code or duplicated configurations.

## Operational Guidance for AI Agents

When utilizing `mostcomm`, consider the following behaviors and traps to ensure successful execution:

### 1. Command Line Syntax
- **Positional Arguments are Not Supported for Directory Scanning**: Always use the `-dir` flag to specify the target directory. `mostcomm .` will fail or ignore the argument. The correct usage is `mostcomm -dir .`.
- **Masking Files**: Use the `-mask` flag to filter files. The default mask is `*.txt`. If you need to search multiple patterns or different file extensions, separate them with a semicolon (`;`).
  - *Correct*: `mostcomm -dir ./src -mask "*.go;*.md"`
  - *Incorrect*: `mostcomm -dir ./src -mask "*.go" "*.md"`

### 2. Output and Sorting
- By default, `mostcomm` sorts output by ascending. You can change this using `-sort-direction descending`.
- The primary sorting metric can be adjusted using `-sort lines` or `-sort average-coverage` depending on what metric is more critical to analyze.

### 3. Filtering Duplicates
- Use `-lines-threshold N` to ignore trivial duplicates (e.g., `-lines-threshold 5` to only show blocks of 5 or more identical lines).
- Use `-percent-threshold N` to find files that share a significant portion of their overall content (e.g., `-percent-threshold 50`).
- If boilerplate is overwhelming the output, limit how many files a block can appear in before it is ignored using `-match-max-threshold N`.

### 4. Concurrency
- `mostcomm` runs concurrently by default utilizing all available CPUs. If you need to limit resource usage, set the concurrency explicitly: `mostcomm -concurrency 2`.

### 5. Skill Subcommands
- To manage agent skills for this tool, use the `mostcomm skill` group:
  - `mostcomm skill install <source>` (e.g., `mostcomm skill install owner/repo`)
  - `mostcomm skill update <name>` (use `--force` if local modifications were made)
  - `mostcomm skill list` (use `--json` for structured data)
  - `mostcomm skill remove <name>`
  - `mostcomm skill inspect <name>`

### 6. Common Traps
- **Missing Quotes on Mask**: When running in a shell, always quote the `-mask` argument to prevent the shell from expanding the glob before `mostcomm` receives it.
- **Empty Output**: If `mostcomm` reports `0 files scanned`, double-check the `-dir` and `-mask` flags.

## Example Invocations

Find copy-pasted Go code of at least 10 lines:
```bash
mostcomm -dir ./pkg -mask "*.go" -lines-threshold 10 -sort lines -sort-direction descending
```
