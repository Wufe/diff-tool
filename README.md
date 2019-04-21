# Diff tool

Lines diff tool tuned for performance.

### Features

+ Multithreading structure
+ Dichotomic search
+ Simplicity

### Useful for

Given two files:  
- looks for lines from file 1 that appear between file 2's lines **or**
- looks for lines from file 1 that do not appear between file 2's lines

### Usage

`diff <file1> <file2> [--left --intersection --dichotomic --output <outputfile> --parallel <number> --threads <number> --ignore-case --left-contains-right --right-contains-left --verbose --stdout --help]`