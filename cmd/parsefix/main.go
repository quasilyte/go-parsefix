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

	"github.com/pkg/errors"
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

	src := newSrcFile(data)
	issues := flag.Args()
	if len(issues) == 0 {
		issues = collectParseErrors(argv.filename, data)
		// It there're still no issues, do an early exit with special exit code.
		if len(issues) == 0 {
			return nothingToFixExit, nil
		}
	}

	fixedAnything := false

	// TODO(quasilyte): this fixer can be done better,
	// but this implementation will do for now.
	funcOpenBraceFixer := fixer{
		match: func(ctx *fixerContext) bool {
			const errorPat = `expected declaration, found '{'`
			return strings.Contains(ctx.issue, errorPat) &&
				ctx.prevLineContains("func ")
		},
		repair: func(ctx *fixerContext) {
			ctx.prevLineReplace("\n", "{\n")
			ctx.replace("{", "")
		},
	}

	// List of all defined fixers.
	fixers := []fixer{
		funcOpenBraceFixer,

		missingByteFixer(
			`missing ',' before newline in composite literal`,
			','),

		missingByteFixer(
			`missing ',' in composite literal`,
			','),

		missingByteFixer(
			`missing ',' in argument list`,
			','),

		missingByteFixer(
			`missing ',' in parameter list`,
			','),

		missingByteFixer(
			`expected ':', found newline`,
			':'),

		missingByteFixer(
			`expected ';', found `, ';'),

		removeCaptureFixer(
			`illegal character U\+[0-9A-F]+ '(.)'`),

		removeCaptureFixer(
			`expected statement, found '(.*)'`),

		replacingFixer(
			`expected boolean or range expression, found assignment`,
			`:= `,
			`:= range `),
	}

	// Try to fix as much issues as possible.
	//
	// Some parsing errors may cause more than one error, but are fixed
	// by a single change. This is why exiting "successfully" when resolved
	// less than len(issues) errors makes sense.
	for _, issue := range issues {
		m := errorPrefixRE.FindStringSubmatch(issue)
		if m == nil {
			continue
		}
		loc := locationInfo(m)
		if loc.file != argv.filename {
			continue
		}
		if tryFix(src, loc, fixers, issue) {
			fixedAnything = true
		}
	}

	if !fixedAnything {
		return fixedNoneExit, nil
	}

	if argv.inplace {
		if err := ioutil.WriteFile(argv.filename, src.Bytes(), 0644); err != nil {
			return errorExit, errors.Errorf("write inplace: %v", err)
		}
	} else {
		argv.w.Write(src.Bytes())
	}

	return fixedSomeExit, nil
}

// errorPrefixRE is an anchor that we expect to see at the beginning of every parse error.
// It captures filename, source line and column.
var errorPrefixRE = regexp.MustCompile(`(.*):(\d+):(\d+): `)

func locationInfo(match []string) location {
	// See `errorPrefixRE`.
	return location{
		file: match[1],

		// Substract 1 from column, so we have a proper index
		// into a line slice, but don't substract 1 from the
		// line, since we have a sentinel line entry in
		// the beginning.
		line:   atoi(match[2]),
		column: atoi(match[3]) - 1,
	}
}

// location is decoded source code position.
// TODO(quasilyte): use `token.Position`?
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

type fixer struct {
	match  func(*fixerContext) bool
	repair func(*fixerContext)
}

func removeCaptureFixer(errorPat string) fixer {
	re := regexp.MustCompile(errorPat)
	var m []string
	return fixer{
		match: func(ctx *fixerContext) bool {
			m = re.FindStringSubmatch(ctx.issue)
			return m != nil
		},
		repair: func(ctx *fixerContext) {
			ctx.replace(m[1], "")
		},
	}
}

func replacingFixer(errorPat, from, to string) fixer {
	return fixer{
		match: func(ctx *fixerContext) bool {
			return strings.Contains(ctx.issue, errorPat) &&
				ctx.contains(from)
		},
		repair: func(ctx *fixerContext) {
			ctx.replace(from, to)
		},
	}
}

func missingByteFixer(errorPat string, toInsert byte) fixer {
	return fixer{
		match: func(ctx *fixerContext) bool {
			return strings.Contains(ctx.issue, errorPat)
		},
		repair: func(ctx *fixerContext) {
			ctx.insertByte(toInsert)
		},
	}
}

func tryFix(src *srcFile, loc location, fixers []fixer, issue string) bool {
	ctx := &fixerContext{
		issue: issue,
		loc:   loc,
		src:   src,
	}
	for _, fix := range fixers {
		if fix.match(ctx) {
			fix.repair(ctx)
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
