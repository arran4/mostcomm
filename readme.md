# mostcomm

Finds the most common ranges in multiple files. Algorithm could be smarter, but it does the job.

# Example

Imagine we have these files:

## `a.txt`:
```
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
```

## `b.txt`:
```
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
```

## `c.txt`:
```
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

When we run `mostcomm` with the arguments:
```
mostcomm -dir ./testdata -mask '*.txt'
```

We expect the answer:
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

# FAQ Why?

Why not? I was going to solve this problem with the shell then decided to write something for it. 
