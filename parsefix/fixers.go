package parsefix

import (
	"regexp"
	"strings"
)

type fixer struct {
	match  func(*fixerContext) bool
	repair func(*fixerContext)
}

// fixerList is a list of all defined fixers.
// Initialized inside init() function.
var fixerList []fixer

func init() {
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

	fixerList = []fixer{
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
