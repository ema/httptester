// Copyright (C) 2020 Emanuele Rocca
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenToString(t *testing.T) {
	var tok token
	tok = newToken(STATUS, "200")
	assert.Equal(t, "200", tok.String())

	tok = newToken(INTEGER, "200")
	assert.Equal(t, "INTEGER: 200", tok.String())

	tok = newToken(STRING, "200")
	assert.Equal(t, "STRING: 200", tok.String())

	tok = newToken(ILLEGAL, "200")
	assert.Equal(t, "ILLEGAL token \"200\"", tok.String())

	tok = newToken(EOF, "0")
	assert.Equal(t, "EOF", tok.String())

	tok = newToken(URL_ARG, "-url")
	assert.Equal(t, "-url", tok.String())
}

type scanTest struct {
	input         string
	expectedToken tokenType
	expectedValue string
}

func newScanTest(i string, typ tokenType, val string) scanTest {
	return scanTest{
		input:         i,
		expectedToken: typ,
		expectedValue: val,
	}
}

func TestScan(t *testing.T) {
	s := newScanner(strings.NewReader("# banana potato\n  \n\n handle"))
	tok := s.scan()
	assert.Equal(t, HASH, tok.typ)
	assert.Equal(t, "#", tok.val)
}

func TestScanUseful(t *testing.T) {
	tests := []scanTest{
		newScanTest("", EOF, ""),
		newScanTest(". blah", DOT, "."),
		newScanTest("[", OPEN_BRACKET, "["),
		newScanTest("]", CLOSE_BRACKET, "]"),
		newScanTest("{", OPEN_CURLY, "{"),
		newScanTest("}", CLOSE_CURLY, "}"),
		newScanTest("~", TILDE, "~"),
		newScanTest("$", ILLEGAL, "$"),
		newScanTest("# banana potato\n  \n\n handle", NEWLINE, "\n"),
		newScanTest("# banana potato\n\"ciao\"", STRING, "ciao"),
		newScanTest("# banana", EOF, ""),
		newScanTest("     ", EOF, ""),
		newScanTest("\"", STRING, ""),
		newScanTest("-status-code", ILLEGAL, "-status-code"),
	}

	for _, test := range tests {
		s := newScanner(strings.NewReader(test.input))
		tok := s.ScanUseful()
		assert.Equal(t, test.expectedToken, tok.typ)
		assert.Equal(t, test.expectedValue, tok.val)
	}
}

func TestScanFull(t *testing.T) {
	var tok token

	input := `# Test a basic get request

handle "/endpoint/1" {
    expect req.method eq "GET"
    expect req.headers["User-Agent"] ~ "chrome"
    expect req.body eq ""
    tx -body "Hello world!" -header "X-HTC-Origin: true" -status 200
}

client "nemo" {
    tx -url "/endpoint/1" -method "GET" -header "User-Agent: this might look like chrome to some"
    expect resp.status ne 404
    expect resp.headers["X-Cache"] ~ "miss"
    expect resp.headers["Something-That-Should-Not-Be-Set"] eq ""
}`

	i := 0

	for s := newScanner(strings.NewReader(input)); tok.typ != EOF; tok = s.ScanUseful() {
		i++
	}

	assert.Equal(t, 81, i)
}
