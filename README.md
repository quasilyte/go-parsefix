# go-parsefix

Fixes simple parse errors automatically. Works great in combination with [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports).

## Motivation

Sometimes you miss a trailing comma.<br>
The other time it's a missing `;` or `}`.<br>
If you're a beginner, you'll probably put `{` on a wrong line several times,<br>
breaking the `gofmt` due to the parsing errors.

Stop interrupting yourself with such nuisances!  
Let `parsefix` perform it's magic.

You do `ctrl+S` in your favourite IDE/editor, it tries to do `gofmt` (or `goimports`), which fails due
to parsing errors, then plugin invokes `parsefix`, which could fix all those issues so `gofmt`
can be executed again successfully. In the end, you don't even notice that there were a minor parsing
error at all. It's just re-formatted and cleaned up.

**Note**: in bright future we could fix **more** errors, not less, as parsing errors
could be improved in some cases to avoid too vague descriptions that are not
precise enough to perform action with high confidence.

## What can be fixed

`parsefix` does not do anything too smart. It only follows safe suggestions from
error messages that usually lead to fixed source code.

Note that it fixes *parsing* errors, not semantic or type errors.
Sometimes it performs not quite right actions, for example, it could insert a `,` where `:`
would make more sense, but you will notice that in the *typecheck* phase.
The best part is that *typecheck* could actually run over your previously unparsable code.
Type checker usually gives far more precise and concise error messages.

### Fix misplaced opening brace

```go
func f()
{
}
// =>
func f() {
}
```

### Fix missing comma

```go
xs := []string{
	"a"
	"b"
}
// =>
xs := []string{
	"a",
	"b",
}

xs := []int{1 2}
// =>
xs := []int{1, 2}

foo(1 2)
// =>
foo(1, 2)

func f(a int b int32) {}
// =>
func f(a int, b int32) {}
```

### Fix missing colon

```go
switch v {
case 1
	return a
case 2
	return b
}
// =>
switch v {
case 1:
	return a
case 2:
	return b
}
```

### Fix missing semicolon

```go
x++ y++
// =>
x++; y++

if x := 1 x != y {
}
// =>
if x := 1; x != y {
}
```

### Fix misplaced tokens

```go
func f() {
	:=
	g()
}
// =>
func f() {
	g()
}
```

### Fix illegal characters

```go
func f() {
	$ g()
	🔥 g()
	# g()
	№ g()
}
// =>
func f() {
	g()
	g()
	g()
	g()
}
```
## Problems

Some parsing errors drive Go parser mad.  
A single typo causes multiple parts of the source file to issue parsing errors.  
This could lead to false-positives, unfortunately.

It would also be great to fix things like `var x := y` to `var x = y`, but
there is no way to do so currently, as messages for this kinds of errors are ambiguous and
do not mention `:=` at all.

Maybe `parsefix` will learn to avoid those in future.
We need more user feedback and bug reports.

## Integration

For ease of integration there are two modes:

1. Accept (full) filename, parse file, try to fix errors that are found during parsing.
2. Accept (full) filename + list of parsing errors, try to fix all provided errors. This is useful if you already have parsing errors and want to speedup things a little bit (avoids re-parsing). Filename is used to filter errors. Caller may provide errors for multiple files, but only those that match filename will be addressed.

Fixed file contents are printed to `stdout` by default.  
Flag `-i` causes `parsefix` to overwrite `-f` contents.

Exit code:
* 0 if at least one issue was fixed.
* 1 and no output if no issues were fixed. With `-i` no re-write is done.
* 2 and no output if there were no parsing issues at all. With `-i` no re-write is done.

Examples:
```bash
# Uses 1st mode. parsefix does parsing itself and prints fixed code to stdout.
parsefix -f=foo/main.go

# Uses 2nd mode. No parsing is done by parsefix.
parsefix -f=foo/main.go "foo/main.go:9:3: expected '{', found 'EOF'"
```
