package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/quasilyte/go-parsefix/parsefix"
)

type exitCode int

// Exit codes are part of the parsefix public API.
//go:generate stringer -type=exitCode
const (
	// fixedSomeExit is used when there were some parsing errors
	// and at least one of them is fixed.
	// Input file is overwritten if -inplace is set to true.
	fixedSomeExit exitCode = 0

	// fixedNoneExir is used when there were some parsing errors
	// and none of them are fixed.
	// Input file is not affected even if -inplace is set to true.
	fixedNoneExit exitCode = 1

	// exitNothingToFix is used when there were no parsing errors.
	// Input file is not affected even if -inplace is set to true.
	nothingToFixExit exitCode = 2

	// errorExit is used when parsefix failed to execute its duties.
	// Input file is not affected even if -inplace is set to true.
	errorExit exitCode = 3
)

type arguments struct {
	filename string
	inplace  bool
	issues   []string
	w        io.Writer
}

func main() {
	log.SetFlags(0)

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `usage: parsefix -f=path/file.go\n`)
		fmt.Fprint(os.Stderr, `usage: parsefix -f=path/file.go "path/file.go:1:2: error text"...`)
		flag.PrintDefaults()
	}

	var argv arguments
	flag.StringVar(&argv.filename, "f", "",
		`full file name for file being fixed`)
	flag.BoolVar(&argv.inplace, "inplace", false,
		`write updated contents to -f instead of stdout`)
	flag.Parse()

	if argv.filename == "" {
		log.Printf("error: provide non-empty -f (filename) option")
		os.Exit(int(errorExit))
	}

	argv.w = os.Stdout

	code, err := runMain(&argv)
	if err != nil {
		log.Printf("error: %v", err)
		os.Exit(int(code))
	}
	os.Exit(int(code))
}

func runMain(argv *arguments) (exitCode, error) {
	data, err := ioutil.ReadFile(argv.filename)
	if err != nil {
		return errorExit, errors.Errorf("read file: %v", err)
	}

	issues := flag.Args()
	if len(issues) == 0 {
		issues = collectParseErrors(argv.filename, data)
		// It there're still no issues, do an early exit with special exit code.
		if len(issues) == 0 {
			return nothingToFixExit, nil
		}
	}

	repaired, err := parsefix.Repair(data, argv.filename, issues)
	if err != nil {
		return errorExit, err
	}
	if repaired == nil {
		return fixedNoneExit, nil
	}

	if argv.inplace {
		if err := ioutil.WriteFile(argv.filename, repaired, 0644); err != nil {
			return errorExit, errors.Errorf("write inplace: %v", err)
		}
	} else {
		argv.w.Write(repaired)
	}

	return fixedSomeExit, nil
}

func collectParseErrors(filename string, src []byte) []string {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filename, src, 0)
	if err == nil {
		return nil
	}
	list := err.(scanner.ErrorList)
	lines := make([]string, len(list))
	for i := range list {
		lines[i] = list[i].Error()
	}
	return lines
}
