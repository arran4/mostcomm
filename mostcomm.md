% mostcomm(1) mostcomm | June 2025
# NAME
mostcomm - detect duplicate blocks of lines across files

# SYNOPSIS
**mostcomm** [**-dir** *DIRECTORY*] [**-mask** *PATTERN*] [**-sort** *STYLE*] [**-sort-direction** *DIRECTION*] [**-percent-threshold** *N*] [**-lines-threshold** *N*] [**-match-max-threshold** *N*]

# DESCRIPTION
**mostcomm** scans a directory of text files and reports consecutive lines that appear in more than one file. It can help locate copy-pasted sections, repeated configuration and other duplicated content. The command hashes each line and builds ranges shared between files. Results may be filtered and sorted with the flags below. Each result lists the file names and line ranges containing the duplicated block along with the run length.

# OPTIONS
* **-dir** *DIRECTORY*: Directory to scan. Defaults to `.`.
* **-mask** *PATTERN*: Semicolon separated glob patterns for files to scan. The default is `*.txt`.
* **-sort** *STYLE*: Sorting algorithm: `none`, `lines` or `average-coverage`.
* **-sort-direction** *DIRECTION*: Either `ascending` or `descending`. The default is ascending.
* **-percent-threshold** *N*: Only report duplicates covering at least *N* percent of a file.
* **-lines-threshold** *N*: Only report duplicates of at least *N* lines.
* **-match-max-threshold** *N*: Exclude duplicates that appear in more than *N* files. Use a negative value to disable.

# EXAMPLES
Run against the bundled sample data:

```
mostcomm -dir ./testdata -mask '*.txt'
```

Only show duplicates of at least five lines that appear in three or more files:

```
mostcomm -dir src -mask '*.go' -lines-threshold 5 -match-max-threshold 3
```

Sort by line count descending:

```
mostcomm -dir logs -mask '*.log' -sort lines -sort-direction descending
```

# SEE ALSO
The project README provides further documentation and examples.
<https://github.com/arran4/mostcomm/>

# AUTHOR
Arran Ubels. Licensed under the MIT License.
