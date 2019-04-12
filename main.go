package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Parallelism represents the number of parallel search process
// const Parallelism = 10

// Threads represents the number of physical threads used by the program
// const Threads = 5

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	conf := parseCLI()

	runtime.GOMAXPROCS(conf.Threads)

	if conf.Verbose {
		fmt.Println("Reading files..")
	}

	firstFilePath, err := filepath.Abs(os.Args[1])
	check(err)
	secondFilePath, err := filepath.Abs(os.Args[2])
	check(err)

	firstFileContent, err := ioutil.ReadFile(firstFilePath)
	check(err)
	secondFileContent, err := ioutil.ReadFile(secondFilePath)
	check(err)

	firstLines := strings.Split(string(firstFileContent), "\n")
	secondLines := strings.Split(string(secondFileContent), "\n")

	if conf.Verbose {
		fmt.Println(fmt.Sprintf("First file lines: %d", len(firstLines)))
		fmt.Println(fmt.Sprintf("Second file lines: %d", len(secondLines)))
	}

	queueChan := make(chan struct{}, conf.Parallel)

	if conf.IgnoreCase || conf.Clean {
		fmt.Println("Cleaning..")
		for index := range firstLines {
			if conf.IgnoreCase {
				firstLines[index] = strings.ToLower(firstLines[index])
			}
			if conf.Clean {
				firstLines[index] = strings.TrimSpace(firstLines[index])
			}
		}
		for index := range secondLines {
			if conf.IgnoreCase {
				secondLines[index] = strings.ToLower(secondLines[index])
			}
			if conf.Clean {
				secondLines[index] = strings.TrimSpace(secondLines[index])
			}
		}
	}

	if conf.Dichotomic {
		if conf.Verbose {
			fmt.Println("Sorting..")
		}
		sort.Strings(secondLines)
	}

	results := startQueue(&conf, queueChan, &firstLines, &secondLines)

	if conf.Stdout {
		for _, line := range results {
			fmt.Println(line)
		}
	}
}

func startQueue(confPtr *cliConf, queueChan chan struct{}, firstPtr *([]string), secondPtr *([]string)) []string {
	if confPtr.Verbose {
		fmt.Println("Starting..")
	}

	// initializing the queue
	for i := 0; i < confPtr.Parallel; i++ {
		queueChan <- struct{}{}
	}

	firstIndex := 0

	resultList := []string{}

	searchResultChan := make(chan string)
	var fileHandle *os.File
	go func() {
		if confPtr.Output != "" {
			var err error
			fileHandle, err = os.Create(confPtr.Output)
			if err != nil {
				panic(err)
			}
		}
		for {
			select {
			case searchResultLine := <-searchResultChan:
				if confPtr.Output != "" {
					fileHandle.WriteString(searchResultLine + "\n")
				}
				resultList = append(resultList, searchResultLine)
			default:
			}
		}
	}()

	// Enables waiting for every working thread to have finished before exiting the processing
	var wg sync.WaitGroup

	firstLength := len(*firstPtr)

L:
	for {
		select {
		case <-queueChan:
			if firstIndex < firstLength {
				wg.Add(1)
				go searchLine(firstIndex, firstPtr, secondPtr, searchResultChan, queueChan, &wg, confPtr)
				firstIndex++
			} else {
				break L
			}
		default:
			if firstIndex == firstLength {
				break L
			}
		}
	}

	// Wait for every working thread to finish
	wg.Wait()

	// Waiting to let the result list to receive or dump the message from the last thread
	time.Sleep(200 * time.Millisecond)

	if confPtr.Output != "" {
		fileHandle.Close()
	}

	return resultList
}

func searchLine(firstIndex int, firstPtr *([]string), secondPtr *([]string), returnChan chan string, queueChan chan struct{}, wg *sync.WaitGroup, confPtr *cliConf) {
	firstLine := (*firstPtr)[firstIndex]
	found := false
	if confPtr.Dichotomic {
		// if confPtr.IgnoreCase {
		firstLine = strings.ToLower(firstLine)
		// }
		foundIndex := sort.SearchStrings(*secondPtr, firstLine)
		if foundIndex >= len(*secondPtr) || (*secondPtr)[foundIndex] != firstLine {
			found = false
		} else {
			found = true
		}
	} else {
		for _, secondLine := range *secondPtr {
			left := firstLine
			right := secondLine
			// if confPtr.IgnoreCase {
			// 	left = strings.ToLower(left)
			// 	right = strings.ToLower(right)
			// }
			if !confPtr.LeftContainsRight && !confPtr.RightContainsLeft {
				if left == right {
					found = true
					break
				}
			} else if confPtr.LeftContainsRight && !confPtr.RightContainsLeft {
				if strings.Contains(left, right) {
					found = true
					break
				}
			} else if !confPtr.LeftContainsRight && confPtr.RightContainsLeft {
				if strings.Contains(right, left) {
					found = true
					break
				}
			} else if confPtr.LeftContainsRight && confPtr.RightContainsLeft {
				if strings.Contains(left, right) || strings.Contains(right, left) {
					found = true
					break
				}
			}
		}
	}
	if confPtr.Intersection {
		if found {
			returnChan <- firstLine
		}
	} else if confPtr.Left {
		if !found {
			returnChan <- firstLine
		}
	}

	queueChan <- struct{}{}
	wg.Done()
}

type cliConf struct {
	First             string
	Second            string
	Output            string
	Dichotomic        bool
	Parallel          int
	Threads           int
	IgnoreCase        bool
	LeftContainsRight bool
	RightContainsLeft bool
	Verbose           bool
	Stdout            bool
	Clean             bool
	Left              bool
	Intersection      bool
}

func parseCLI() cliConf {

	conf := cliConf{
		Output:            "",
		Dichotomic:        false,
		Threads:           5,
		Parallel:          10,
		IgnoreCase:        false,
		LeftContainsRight: false,
		RightContainsLeft: false,
		Verbose:           false,
		Stdout:            false,
		Clean:             true,
		Left:              true,
		Intersection:      false,
	}

	if len(os.Args) < 3 {
		printDefaultCLI(&conf)
	}

	conf.First = os.Args[1]
	conf.Second = os.Args[2]

	for index, arg := range os.Args {
		if arg == "--output" {
			if len(os.Args) < index+2 {
				printDefaultCLI(&conf)
			} else {
				conf.Output = os.Args[index+1]
			}
		}
		if arg == "--parallel" {
			if len(os.Args) < index+2 {
				printDefaultCLI(&conf)
			} else {
				parallel, err := strconv.Atoi(os.Args[index+1])
				if err != nil {
					printDefaultCLI(&conf)
				}
				conf.Parallel = parallel
			}
		}
		if arg == "--threads" {
			if len(os.Args) < index+2 {
				printDefaultCLI(&conf)
			} else {
				threads, err := strconv.Atoi(os.Args[index+1])
				if err != nil {
					printDefaultCLI(&conf)
				}
				conf.Threads = threads
			}
		}
		if arg == "--dichotomic" {
			conf.Dichotomic = true
		}
		if arg == "--ignore-case" {
			conf.IgnoreCase = true
		}
		if arg == "--left-contains-right" {
			conf.LeftContainsRight = true
		}
		if arg == "--right-contains-left" {
			conf.RightContainsLeft = true
		}
		if arg == "--verbose" {
			conf.Verbose = true
		}
		if arg == "--stdout" {
			conf.Stdout = true
		}
		if arg == "--left" {
			conf.Left = true
		}
		if arg == "--intersection" {
			conf.Intersection = true
		}

		if arg == "--help" {
			printDefaultCLI(&conf)
		}
	}

	return conf
}

func printDefaultCLI(conf *cliConf) {
	fmt.Println(fmt.Sprintf("\nUsage: %s <file1> <file2> [--left --right --intersection --dichotomic --output <outputfile> --parallel <number> --threads <number> --ignore-case --left-contains-right --right-contains-left --verbose --stdout --help]\n\n", "diff"))
	if conf.Intersection {
		fmt.Println("Looks for lines from file1 that appear between file2's lines.\n")
	} else if conf.Left {
		fmt.Println("Looks for lines from file1 that do not appear between file2's lines.\n")
	}

	fmt.Println("--dichotomic\nUses binary search.\nThe file to be searched against will be sorted.\n\n")
	fmt.Println("--left\nIt is set to true as default.\nReturns left set minus intersection of sets left and right.\n\n")
	fmt.Println("--intersection\nReturns the intersection of sets left and right.\n\n")
	fmt.Println("CURRENT CONFIGURATION")
	fmt.Println(fmt.Sprintf("Parallel processes -> [%d]", conf.Parallel))
	fmt.Println(fmt.Sprintf("Real threads -> [%d]", conf.Threads))
	fmt.Println("")
	os.Exit(0)
}
