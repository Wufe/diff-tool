package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Parallelism represents the number of parallel search process
// const Parallelism = 10

// Threads represents the number of physical threads used by the program
// const Threads = 5

func main() {
	conf := parseCLI()

	runtime.GOMAXPROCS(conf.Threads)

	firstFilePath, _ := filepath.Abs(os.Args[1])
	secondFilePath, _ := filepath.Abs(os.Args[2])

	firstFileContent, _ := ioutil.ReadFile(firstFilePath)
	secondFileContent, _ := ioutil.ReadFile(secondFilePath)

	firstLines := strings.Split(string(firstFileContent), "\n")
	secondLines := strings.Split(string(secondFileContent), "\n")

	queueChan := make(chan struct{}, conf.Parallel)

	results := startQueue(&conf, queueChan, &firstLines, &secondLines)

	for _, line := range results {
		fmt.Println(line)
	}
}

func startQueue(confPtr *cliConf, queueChan chan struct{}, firstPtr *([]string), secondPtr *([]string)) []string {

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
				fileHandle.WriteString(searchResultLine + "\n")
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
	for _, secondLine := range *secondPtr {
		left := firstLine
		right := secondLine
		if confPtr.IgnoreCase {
			left = strings.ToLower(left)
			right = strings.ToLower(left)
		}
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
	if !found {
		returnChan <- firstLine
	}
	queueChan <- struct{}{}
	wg.Done()
}

type cliConf struct {
	First             string
	Second            string
	Output            string
	Dicotomic         bool
	Parallel          int
	Threads           int
	IgnoreCase        bool
	LeftContainsRight bool
	RightContainsLeft bool
}

func parseCLI() cliConf {

	conf := cliConf{
		Output:            "",
		Dicotomic:         false,
		Threads:           5,
		Parallel:          10,
		IgnoreCase:        false,
		LeftContainsRight: false,
		RightContainsLeft: false,
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
		if arg == "--dicotomic" {
			conf.Dicotomic = true
		}
		if arg == "--help" {
			printDefaultCLI(&conf)
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
	}

	return conf
}

func printDefaultCLI(conf *cliConf) {
	fmt.Println(fmt.Sprintf("\nUsage: %s <file1> <file2> [--dicotomic --output <outputfile> --parallel <number> --threads <number> --ignore-case --left-contains-right --right-contains-left]\n\nLooks for lines from file1 that do not appear between file2's lines.\n", "diff"))
	fmt.Println("CURRENT CONFIGURATION")
	fmt.Println(fmt.Sprintf("Parallel processes -> [%d]", conf.Parallel))
	fmt.Println(fmt.Sprintf("Real threads -> [%d]", conf.Threads))
	fmt.Println("")
	os.Exit(0)
}
