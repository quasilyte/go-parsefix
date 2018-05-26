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
	"regexp"
	"strconv"
	"strings"
)

// Exit codes are part of the parsefix public API.
const (
	// exitFixedSome is used when there were some parsing errors
	// and at least one of them is fixed.
	exitFixedSome = 0

	// exitFixedNone is used when there were some parsing errors
	// and none of them is fixed.
	exitFixedNone = 1

	// exitNothingToFix is used when there were no parsing errors.
	exitNothingToFix = 2
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `usage: parsefix -f=path/file.go\n`)
		fmt.Fprint(os.Stderr, `usage: parsefix -f=path/file.go "path/file.go:1:2: error text"...`)
		flag.PrintDefaults()
	}
	filename := flag.String("f", "",
		`full file name for file being fixed`)
	inplace := flag.Bool("inplace", false,
		`write updated contents to -f instead of stdout`)
	flag.Parse()

	if *filename == "" {
		log.Fatalf("provide non-empty -filename option")
	}

	data, err := ioutil.ReadFile(*filename)
	if err != nil {
		log.Fatal(*filename)
	}
	src := byteVector(data)
	issues := flag.Args()
	if len(issues) == 0 {
		issues = collectParseErrors(*filename, data)
		// It there're still no issues, do an early exit with special exit code.
		if len(issues) == 0 {
			os.Exit(exitNothingToFix)
		}
	}

	fixedAnything := false

	// List of all defined fixers.
	fixers := []fixer{
		missingByteFixer(`missing ',' before newline in composite literal`, ','),
		missingByteFixer(`missing ',' in composite literal`, ','),
		missingByteFixer(`missing ',' in argument list`, ','),
		missingByteFixer(`missing ',' in parameter list`, ','),
		missingByteFixer(`expected ':', found newline`, ':'),
		missingByteFixer(`expected ';', found `, ';'),
		removeCaptureFixer(`illegal character U\+[0-9A-F]+ '(.)'`),
		removeCaptureFixer(`expected statement, found '(.*)'`),
	}

	// Try to fix as much issues as possible.
	// Some parsing errors may cause more than one error, but are fixed
	// by a single change. This is why exiting "successfully" when resolved
	// less than len(issues) errors makes sense.
	for _, issue := range issues {
		m := errorPrefixRE.FindStringSubmatch(issue)
		if m == nil {
			continue
		}
		loc := locationInfo(m)
		if loc.file != *filename {
			continue
		}
		if tryFix(&src, loc, fixers, issue) {
			fixedAnything = true
		}
	}

	if !fixedAnything {
		os.Exit(exitFixedNone)
	}

	var dst io.Writer
	if *inplace {
		f, err := os.OpenFile("test.txt", os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			panic(err)
		}
		dst = f
	} else {
		dst = os.Stdout
	}
	if _, err := dst.Write([]byte(src)); err != nil {
		panic(err)
	}
	os.Exit(exitFixedSome)
}

// errorPrefixRE is an anchor that we expect to see at the beginning of every parse error.
// It captures filename, source line and column.
var errorPrefixRE = regexp.MustCompile(`(.*):(\d+):(\d+): `)

func locationInfo(match []string) location {
	// See errorPrefix.
	return location{
		file:   match[1],
		line:   atoi(match[2]),
		column: atoi(match[3]),
	}
}

// location is decoded source code position.
type location struct {
	file   string
	line   int
	column int
}

// atoi is like strconv.Atoi, but panics on errors.
// We're using it to decode source code locations: columns and line numbers,
// if they are not valid numbers, it's very dread situation.
func atoi(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return v
}

// byteVector is a convenience wrapper around []byte that makes
// content updating operations easier.
type byteVector []byte

// InsertByteAt inserts b into v at specified position.
func (v *byteVector) InsertByteAt(b byte, pos int) {
	*v = append(*v, 0)
	copy((*v)[pos+1:], (*v)[pos:])
	(*v)[pos] = b
}

// Locate finds source offset for location.
func (v *byteVector) Locate(loc location) int {
	data := *v

	p := 0
	line := 1
	column := 1
	for p < len(data) {
		switch data[p] {
		case '\n':
			line++
			column = 1
		default:
			column++
		}
		p++
		if line == loc.line && column == loc.column {
			return p
		}
	}
	return len(data) - 1
}

type fixer struct {
	match  func(s string) bool
	repair func(src *byteVector, loc location)
}

func removeCaptureFixer(errorPat string) fixer {
	re := regexp.MustCompile(errorPat)
	var m []string
	return fixer{
		match: func(s string) bool {
			m = re.FindStringSubmatch(s)
			return m != nil
		},
		repair: func(src *byteVector, loc location) {
			pos := src.Locate(loc)
			for i := 0; i < len(m[1]); i++ {
				(*src)[pos+i] = ' '
			}
		},
	}
}

func missingByteFixer(errorPat string, toInsert byte) fixer {
	return fixer{
		match: func(s string) bool {
			return strings.Contains(s, errorPat)
		},
		repair: func(src *byteVector, loc location) {
			pos := src.Locate(loc)
			src.InsertByteAt(toInsert, pos)
		},
	}
}

func tryFix(src *byteVector, loc location, fixers []fixer, issue string) bool {
	for _, fix := range fixers {
		if fix.match(issue) {
			fix.repair(src, loc)
			return true
		}
	}
	return false
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
