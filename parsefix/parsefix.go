package parsefix

import (
	"regexp"
	"strconv"
)

// Repair tries to fix parsing issues inside code.
// Issues are filtered by using filename.
//
// Returns nil if no issues were fixed.
// Returns non-nil byte slice of repaired code otherwise.
func Repair(code []byte, filename string, issues []string) ([]byte, error) {
	src := newSrcFile(code)

	fixedAnything := false

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
		if loc.file != filename {
			continue
		}
		if tryFix(src, loc, issue) {
			fixedAnything = true
		}
	}

	if fixedAnything {
		return src.Bytes(), nil
	}
	return nil, nil
}

func tryFix(src *srcFile, loc location, issue string) bool {
	ctx := &fixerContext{
		issue: issue,
		loc:   loc,
		src:   src,
	}
	for _, fix := range fixerList {
		if fix.match(ctx) {
			fix.repair(ctx)
			return true
		}
	}
	return false
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
