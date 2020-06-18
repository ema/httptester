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

// The Httptester Test Case (HTC) language scanner
//
// Basic scanner structure idea from:
// https://github.com/benbjohnson/sql-parser

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

type tokenType int

const (
	// Special tokens
	ILLEGAL tokenType = iota
	EOF
	WS

	// Literals
	STRING  // header names and values, method names, ...
	INTEGER // status codes, Content-Length, ...

	// Misc characters
	DOT           // .
	HASH          // #
	OPEN_BRACKET  // [
	CLOSE_BRACKET // ]
	OPEN_CURLY    // {
	CLOSE_CURLY   // }

	// Operators
	EQUAL    // eq
	NOTEQUAL // ne
	TILDE    // ~

	// Keywords
	HANDLE // handle
	CLIENT // client
	EXPECT // expect
	TX     // tx
	// Request/response HTTP info like eg: resp.status, req.headers
	REQ     // req
	RESP    // resp
	METHOD  // method
	STATUS  // status
	HEADERS // headers
	BODY    // body

	// Arguments
	BODY_ARG   // -body
	STATUS_ARG // -status
	HEADER_ARG // -header
	URL_ARG    // -url
	METHOD_ARG // -method
)

// token represents a lexical token. eg: {typ:STATUS val:"200"}
type token struct {
	typ tokenType
	val string
}

func newToken(t tokenType, v string) token {
	return token{typ: t, val: v}
}

// Pretty-print a token
func (t token) String() string {
	switch t.typ {
	case ILLEGAL:
		return fmt.Sprintf("ILLEGAL token %q", t.val)
	case STRING:
		return fmt.Sprintf("STRING: %s", t.val)
	case INTEGER:
		return fmt.Sprintf("INTEGER: %s", t.val)
	case EOF:
		return "EOF"
	}

	return t.val
}

// Scanner represents a lexical scanner
type scanner struct {
	r *bufio.Reader
}

func newScanner(r io.Reader) *scanner {
	return &scanner{r: bufio.NewReader(r)}
}

// scan returns the next token
func (s *scanner) scan() token {
	// Read the next rune
	ch := s.read()

	if isWhitespace(ch) {
		// whitespace, consume all contiguous whitespace
		s.unread()
		return s.scanWhitespace()
	} else if isLetter(ch) || isDigit(ch) || ch == '-' || ch == '_' {
		// letter/digit, consume as an ident or reserved word
		s.unread()
		return s.scanIdent()
	} else if ch == '"' {
		// Quoted string, read till closing '#'
		return s.scanQuotedString()
	} else if ch == '#' {
		// comment, read till newline or EOF
		for {
			ch = s.read()
			if ch == '\n' {
				return newToken(HASH, "#")
			}

			if ch == eof {
				break
			}
		}
	}

	// Otherwise read the individual character.
	switch ch {
	case eof:
		return newToken(EOF, "")
	case '.':
		return newToken(DOT, string(ch))
	case '[':
		return newToken(OPEN_BRACKET, string(ch))
	case ']':
		return newToken(CLOSE_BRACKET, string(ch))
	case '{':
		return newToken(OPEN_CURLY, string(ch))
	case '}':
		return newToken(CLOSE_CURLY, string(ch))
	case '~':
		return newToken(TILDE, string(ch))
	}

	return newToken(ILLEGAL, string(ch))
}

// ScanUseful returns the next non-whitespace, non-comment token
func (s *scanner) ScanUseful() token {
	for {
		t := s.scan()
		if t.typ != WS && t.typ != HASH {
			return t
		}
	}
}

// scanWhitespace consumes the current rune and all contiguous whitespace
func (s *scanner) scanWhitespace() token {
	for {
		ch := s.read()

		if ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		}
	}

	return newToken(WS, " ")
}

func (s *scanner) scanQuotedString() token {
	var buf bytes.Buffer

	// Read every subsequent character into the buffer, and stop as soon as a
	// closing " is found. EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if ch == '"' {
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	return newToken(STRING, buf.String())
}

// scanIdent consumes the current rune and all contiguous ident runes
func (s *scanner) scanIdent() token {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent ident character into the buffer.
	// Non-ident characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isLetter(ch) && !isDigit(ch) && ch != '-' && ch != '_' {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	str := buf.String()

	// If the string matches a keyword then return that keyword
	switch str {
	case "eq":
		return newToken(EQUAL, str)
	case "ne":
		return newToken(NOTEQUAL, str)
	case "handle":
		return newToken(HANDLE, str)
	case "client":
		return newToken(CLIENT, str)
	case "expect":
		return newToken(EXPECT, str)
	case "req":
		return newToken(REQ, str)
	case "resp":
		return newToken(RESP, str)
		// req/resp fields follow
	case "method":
		return newToken(METHOD, str)
	case "headers":
		return newToken(HEADERS, str)
	case "body":
		return newToken(BODY, str)
	case "status":
		return newToken(STATUS, str)
	case "tx":
		return newToken(TX, str)
		// tx arguments follow
	case "-body":
		return newToken(BODY_ARG, str)
	case "-status":
		return newToken(STATUS_ARG, str)
	case "-header":
		return newToken(HEADER_ARG, str)
	case "-method":
		return newToken(METHOD_ARG, str)
	case "-url":
		return newToken(URL_ARG, str)
	}

	if _, err := strconv.Atoi(str); err == nil {
		// Looks like an integer
		return newToken(INTEGER, str)
	}

	// Otherwise assume this is illegal
	return newToken(ILLEGAL, str)
}

// read reads the next rune from the buffered reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (s *scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *scanner) unread() { _ = s.r.UnreadRune() }

// isWhitespace returns true if the rune is a space, tab, or newline.
func isWhitespace(ch rune) bool { return ch == ' ' || ch == '\t' || ch == '\n' }

// isLetter returns true if the rune is a letter.
func isLetter(ch rune) bool { return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') }

// isDigit returns true if the rune is a digit.
func isDigit(ch rune) bool { return (ch >= '0' && ch <= '9') }

// eof represents a marker rune for the end of the reader.
var eof = rune(0)
