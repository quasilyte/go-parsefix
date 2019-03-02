package parsefix

import (
	"bytes"
)

type fixerContext struct {
	issue string
	loc   location
	src   *srcFile
}

func (ctx *fixerContext) prevLineContains(s string) bool {
	return bytes.Contains(ctx.src.lines[ctx.loc.line-1], []byte(s))
}

func (ctx *fixerContext) contains(s string) bool {
	return bytes.Contains(ctx.src.lines[ctx.loc.line], []byte(s))
}

func (ctx *fixerContext) prevLineReplace(from, to string) {
	ctx.src.lines[ctx.loc.line-1] = bytes.Replace(
		ctx.src.lines[ctx.loc.line-1], []byte(from), []byte(to), 1)
}

func (ctx *fixerContext) replace(from, to string) {
	ctx.src.lines[ctx.loc.line] = bytes.Replace(
		ctx.src.lines[ctx.loc.line], []byte(from), []byte(to), 1)
}

func (ctx *fixerContext) insertByte(b byte) {
	line := ctx.src.lines[ctx.loc.line]
	pos := ctx.loc.column

	line = append(line, 0)
	copy(line[pos+1:], line[pos:])
	line[pos] = b

	ctx.src.lines[ctx.loc.line] = line
}
