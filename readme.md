# Mostcomm

Mostcomm is a command line tool that scans a directory of text files and reports repeated runs of lines across those files. It is useful for finding copy‑pasted blocks, duplicated configuration or just getting an overview of common data in a large corpus.

The detection algorithm hashes each line and uses those hashes to build ranges that appear in more than one file. Results can be sorted and filtered to help focus on the most significant duplicates.

## Features

- Works on any text files; simply point it at a directory.
- Supports glob patterns (semicolon separated) to select which files are scanned.
- Filter results by a minimum number of lines or a minimum percentage of the file.
- Limit the maximum number of files that may share the same block.
- Sort results by run length or by average file coverage.
- Provides a Go library (`mostcomm` package) so the detection logic can be reused programmatically.

## Installation

If you have Go installed you can build the binary directly from the source repository:

```bash
# clone the repository
git clone https://github.com/arran4/mostcomm.git
cd mostcomm

# install the command
go install ./cmd/mostcomm
```

This will place the `mostcomm` binary in your `GOBIN` (or `$GOPATH/bin`).

## Usage

```
mostcomm -dir DIRECTORY [-mask PATTERN] [options]
```

Common flags:

- `-dir` &nbsp;Directory to scan (default `.`).
- `-mask` &nbsp;Glob patterns to match files (default `*.txt`). Use `;` to separate multiple patterns.
- `-sort` &nbsp;Sorting algorithm for the output (`none`, `lines`, `average-coverage`).
- `-sort-direction` &nbsp;`ascending` or `descending` (default `ascending`).
- `-percent-threshold` &nbsp;Only report duplicates covering at least this percent of a file.
- `-lines-threshold` &nbsp;Only report duplicates of at least this many lines.
- `-match-max-threshold` &nbsp;Exclude duplicates that occur in more than this many files.

### Example

Mostcomm ships with a small example data set under `testdata`. The three text
files contain simple numbered lines so you can easily see what the tool
reports. They are structured as follows:

1. `a.txt` lists the numbers 1 through 10, one per line.
2. `b.txt` contains the same numbers but with a blank line after 5.
3. `c.txt` repeats the range 6 through 10 twice.

```text
# a.txt
1
2
3
4
5
6
7
8
9
10

# b.txt
1
2
3
4
5

6
7
8
9
10

# c.txt
6
7
8
9
10
6
7
8
9
10
```

Run the command against that directory:

```bash
mostcomm -dir ./testdata -mask '*.txt'
```

Which will output something similar to:

```
2023/02/27 22:57:22 Scanning a.txt
2023/02/27 22:57:22 Scanning b.txt
2023/02/27 22:57:22 Scanning c.txt
3 files scanned
11 unique lines scanned
32 total lines scanned
2 Duplicate runs
- a.txt:0-4 (5), b.txt:0-4 (5)
- a.txt:5-9 (5), b.txt:7-11 (5), c.txt:0-4 (5), c.txt:5-9 (5)
```

## Ideas and Contributions

Mostcomm started as a small utility and there is plenty of room for improvement. Some ideas for future enhancements include:

- Smarter range detection that handles reordered lines.
- Output in machine readable formats such as JSON.
- Integration with code editors or CI pipelines.

Pull requests and issues are welcome. Feel free to open a discussion if you have suggestions or run into any problems.

## License

This project is released under the [MIT License](LICENSE).
