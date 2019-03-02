package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParsefix(t *testing.T) {
	tests, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("list `testdata` dir: %v", err)
	}

	runTest := func(t *testing.T, testName string) {
		dir := filepath.Join("testdata", testName)
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			t.Errorf("list `%s` dir: %v", dir, err)
			return
		}
		badFiles := map[string]string{}
		goodFiles := map[string][]byte{}
		for _, file := range files {
			path := filepath.Join(dir, file.Name())
			switch {
			case strings.HasPrefix(file.Name(), "good"):
				data, err := ioutil.ReadFile(path)
				if err != nil {
					t.Errorf("read %s file: %v", file.Name(), err)
					return
				}
				key := strings.TrimPrefix(file.Name(), "good")
				goodFiles[key] = data
			case strings.HasPrefix(file.Name(), "bad"):
				key := strings.TrimPrefix(file.Name(), "bad")
				badFiles[key] = path
			default:
				t.Errorf("file %s has no good/bad prefix", file.Name())
			}
		}

		for key, filename := range badFiles {
			goodData := goodFiles[key]
			if goodData == nil {
				t.Errorf("%s: missing good data", key)
				continue
			}

			var buf bytes.Buffer
			code, err := runMain(&arguments{
				filename: filename,
				w:        &buf,
			})
			switch {
			case err != nil:
				t.Errorf("%s: %v", key, err)
				continue
			case code != fixedSomeExit:
				t.Errorf("%s: unexpected exit code (%v)", key, code)
				continue
			}

			formatted, err := format.Source(buf.Bytes())
			if err != nil {
				t.Errorf("gofmt: %v", err)
				continue
			}

			want := append([]string{""}, strings.Split(string(goodData), "\n")...)
			have := append([]string{""}, strings.Split(string(formatted), "\n")...)
			if diff := cmp.Diff(have, want); diff != "" {
				t.Errorf("%s: output differs from expected (-have +want):\n%s\noutput:\n%s",
					key, diff, string(formatted))
			}
		}
	}

	for _, test := range tests {
		testName := test.Name()
		t.Run(test.Name(), func(t *testing.T) {
			runTest(t, testName)
		})
	}
}
