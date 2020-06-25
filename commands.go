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
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Command is the interface that must be implemented by all commands
// (eg: expect, tx)
type Command interface {
	// Parse fills the command structure by parsing the data in the given
	// scanner, thus implementing the command-specific parsing logic. The
	// returned error is non-nil in case of parse errors
	Parse(*scanner) error
}

// ExpectField represents the various attributes that the expect command can
// take. For example: req.method, resp.status, ...
type ExpectField int

const (
	EXPECT_METHOD ExpectField = iota
	EXPECT_HEADERS
	EXPECT_BODY
	EXPECT_STATUS
)

// Expect is a command used to test a certain assumption. For example, the
// command 'req.method eq "GET"' verifies that the request method is GET, and
// fail if it is not
type Expect struct {
	verbatim   string
	field      ExpectField
	headerName string
	operator   tokenType
	expected   string
}

// String pretty-prints an Expect
func (e Expect) String() string {
	return fmt.Sprintf("%q", e.verbatim)
}

// Parse an expect command. Support both requests (expect req[...]) and
// responses (expect resp[...])
func (e *Expect) Parse(s *scanner) error {
	// Get something like 'req.method'
	token := s.ScanUseful()
	// Start building up e.verbatim
	e.verbatim = token.val
	if token.typ != REQ && token.typ != RESP {
		return fmt.Errorf("Parse error in 'expect' command: expecting {req,resp}, got %q", token)
	}

	token = s.ScanUseful()
	e.verbatim += token.val
	if token.typ != DOT {
		return fmt.Errorf("Parse error in 'expect' command: expecting something like 'req.method', got %q", token)
	}

	token = s.ScanUseful()
	e.verbatim += token.val

	if token.typ == METHOD {
		e.field = EXPECT_METHOD
	} else if token.typ == STATUS {
		e.field = EXPECT_STATUS
	} else if token.typ == BODY {
		e.field = EXPECT_BODY
	} else if token.typ == HEADERS {
		e.field = EXPECT_HEADERS

		// Get header name (open bracket, expect string, close bracket)
		token = s.ScanUseful()
		e.verbatim += token.val
		if token.typ != OPEN_BRACKET {
			return fmt.Errorf("Parse error in 'expect' command: expecting 'req.headers[$hdr_name]', got %q", token)
		}

		token = s.ScanUseful()
		e.verbatim += token.val
		if token.typ != STRING {
			return fmt.Errorf("Parse error in 'expect' command: expecting 'req.headers[$hdr_name]', got %q", token)
		}

		// We've got something looking like a header name
		e.headerName = token.val

		token = s.ScanUseful()
		e.verbatim += token.val
		if token.typ != CLOSE_BRACKET {
			return fmt.Errorf("Parse error in 'expect' command: expecting 'req.headers[$hdr_name]', got %q", token)
		}
	} else {
		return fmt.Errorf("Parse error in 'expect' command: expecting 'req.{method,headers,body}', got %q", token)
	}

	// Get the operator
	token = s.ScanUseful()
	e.verbatim += " " + token.val
	if token.typ != EQUAL && token.typ != NOTEQUAL && token.typ != TILDE {
		return fmt.Errorf("Parse error in 'expect' command: expecting operator to be '{eq,ne,~}', got %q", token)
	}

	// TODO: if token.typ == TILDE, validate regexp with
	// regexp.MustCompile(str)

	e.operator = token.typ

	// Get the value eg: "^(chrome|curl)"
	token = s.ScanUseful()
	e.verbatim += fmt.Sprintf(" %q", token.val)

	if token.typ != STRING && token.typ != INTEGER {
		return fmt.Errorf("Parse error in 'expect' command: expecting a string/integer, got %q", token)
	}

	e.expected = token.val

	return nil
}

// expectThing returns true if what we expect is true given the value of
// 'actual'
func (e Expect) expectThing(actual string) bool {
	switch e.operator {
	case EQUAL:
		return e.expected == actual
	case NOTEQUAL:
		return e.expected != actual
	case TILDE:
		ret, err := regexp.Match(e.expected, []byte(actual))
		if err != nil {
			log.Panic("regexp.Match error: ", err)
		}
		return ret
	}

	log.Panic("Unknown operator: ", e.operator)
	return false
}

// ActualRequest returns the value in the given http.Request object
// corresponding to this Expect. For instance, if we are expecting something
// about the request method, here we return the actual request method sent
func (e Expect) ActualRequest(req http.Request) string {
	var actual string

	switch e.field {
	case EXPECT_METHOD:
		actual = req.Method
	case EXPECT_HEADERS:
		actual = req.Header.Get(e.headerName)
	case EXPECT_BODY:
		if req.Body == nil {
			return ""
		}
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Panic(err)
		} else {
			actual = string(body)
		}
	case EXPECT_STATUS:
		log.Fatal("Requests have no status")
	}

	return actual
}

// Request returns true if the expectations regarding the given request are
// met, false otherwise
func (e Expect) Request(req http.Request) bool {
	return e.expectThing(e.ActualRequest(req))
}

// StringResponse returns a string representation of the given http.Response
func (e Expect) StringResponse(resp http.Response) string {
	s := fmt.Sprintf("HTTP %d\n", resp.StatusCode)
	for key, value := range resp.Header {
		s += fmt.Sprintf("%s: %s\n", key, value)
	}
	return s
}

// ActualResponse returns the value in the given http.Response object
// corresponding to this Expect. For instance, if we are expecting something
// about the response status, here we return the actual response status
func (e Expect) ActualResponse(resp http.Response) string {
	var actual string

	switch e.field {
	case EXPECT_STATUS:
		actual = strconv.Itoa(resp.StatusCode)
	case EXPECT_HEADERS:
		actual = resp.Header.Get(e.headerName)
	case EXPECT_BODY:
		if resp.Body == nil {
			return ""
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Panic(err)
		} else {
			actual = string(body)
		}
	}

	return actual
}

// Response returns true if the expectations regarding the given response are
// met, false otherwise
func (e Expect) Response(resp http.Response) bool {
	return e.expectThing(e.ActualResponse(resp))
}

// TxResp is the command used to make origin servers return an HTTP response.
// An example is:
// tx -body "Hello world!" -header "X-HTC-Origin: true" -status 200
type TxResp struct {
	statusCode int
	headers    map[string]string
	body       string
}

// String pretty-prints a TxResp
func (r TxResp) String() string {
	return fmt.Sprintf("HTTP %d: %q", r.statusCode, r.body)
}

// Parse a tx command in the handle stanza, in other words a response. Eg:
// tx -body "Hello world!" -header "Cache-Control: s-maxage=120" -status 200
func (r *TxResp) Parse(s *scanner) error {
	r.statusCode = 200
	r.headers = make(map[string]string)

	for {
		token := s.ScanUseful()
		if token.typ == EOF || token.typ == CLOSE_CURLY || token.typ == NEWLINE {
			s.unread()
			break
		}
		if token.typ == BODY_ARG {
			token := s.ScanUseful()
			if token.typ != STRING {
				return fmt.Errorf("Parse error in 'tx' command: expecting a string, got %q", token)
			}
			r.body = token.val
		} else if token.typ == HEADER_ARG {
			token := s.ScanUseful()
			if token.typ != STRING {
				return fmt.Errorf("Parse error in 'tx' command: expecting a string, got %q", token)
			}
			splitted := strings.SplitN(token.val, ":", 2)
			if len(splitted) != 2 {
				return fmt.Errorf("Parse error in 'tx' command: expecting a header, got %q", token)
			}
			r.headers[splitted[0]] = splitted[1]
		} else if token.typ == STATUS_ARG {
			token := s.ScanUseful()
			if token.typ != INTEGER {
				return fmt.Errorf("Parse error in 'tx' command: expecting an integer, got %q", token)
			}

			r.statusCode, _ = strconv.Atoi(token.val)
		} else {
			return fmt.Errorf("Parse error in 'tx' command: expecting -body, -header, or -status, got %q", token)
		}
	}

	return nil
}

// Send writes TxResp to the http.ResponseWriter 'writer'
func (r TxResp) Send(writer http.ResponseWriter) bool {
	// Add all headers
	for key, value := range r.headers {
		writer.Header().Add(key, value)
	}
	// Send the status code
	writer.WriteHeader(r.statusCode)

	// Write body
	fmt.Fprintf(writer, r.body)
	return true
}

// TxReq is the command used to make clients send an HTTP request.
// An example is:
// tx -url "/hello/world" -header "X-HTC-Origin: true" -method "HEAD"
type TxReq struct {
	uri     string
	method  string
	headers map[string]string
	body    string
}

// String pretty-prints a TxReq
func (r TxReq) String() string {
	s := fmt.Sprintf("%s %s\n", r.method, r.uri)
	for key, value := range r.headers {
		s += fmt.Sprintf("%s: %s\n", key, value)
	}
	return s
}

// Parse a tx command in the client stanza, in other words a request. Eg:
// tx -url "/endpoint/1" -method "GET" -header "X-Debug: x-cache"
func (r *TxReq) Parse(s *scanner) error {
	r.method = "GET"
	r.headers = make(map[string]string)

	for {
		token := s.ScanUseful()
		// Only "expect" is allowed after "tx" in the client stanza
		if token.typ == EOF || token.typ == CLOSE_CURLY || token.typ == NEWLINE {
			s.unread()
			break
		}
		if token.typ == BODY_ARG {
			token := s.ScanUseful()
			if token.typ != STRING {
				return fmt.Errorf("Parse error in 'tx' command: expecting a string, got %q", token)
			}
			r.body = token.val
		} else if token.typ == HEADER_ARG {
			token := s.ScanUseful()
			if token.typ != STRING {
				return fmt.Errorf("Parse error in 'tx' command: expecting a string, got %q", token)
			}
			splitted := strings.SplitN(token.val, ":", 2)
			if len(splitted) != 2 {
				return fmt.Errorf("Parse error in 'tx' command: expecting a header, got %q", token)
			}
			r.headers[splitted[0]] = splitted[1]
		} else if token.typ == METHOD_ARG {
			token := s.ScanUseful()
			if token.typ != STRING {
				return fmt.Errorf("Parse error in 'tx' command: expecting a string, got %q", token)
			}

			// XXX: check that method isn't "banana"
			r.method = token.val
		} else if token.typ == URL_ARG {
			token := s.ScanUseful()
			if token.typ != STRING {
				return fmt.Errorf("Parse error in 'tx' command: expecting a string, got %q", token)
			}

			// XXX: check that url isn't "banana"
			r.uri = token.val
		} else {
			return fmt.Errorf("Parse error in 'tx' command: expecting -url, -header, method, or -body, got %q", token)
		}
	}

	return nil
}

// Send the TxReq to the given server
func (r TxReq) Send(server string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(r.method, fmt.Sprintf("http://%s%s", server, r.uri), strings.NewReader(r.body))
	if err != nil {
		return nil, err
	}

	// Add all headers
	for key, value := range r.headers {
		req.Header.Add(key, value)
	}

	return client.Do(req)
}
