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

// Httptester Test Case (HTC) language parser

package main

import (
	"fmt"
	"io"
)

type HandleStanza struct {
	URIPath      string
	Expectations []Expect
	Response     TxResp
}

type ClientStanza struct {
	Name         string
	Request      TxReq
	Expectations []Expect
}

func parseHandle(s *scanner) (HandleStanza, error) {
	var h HandleStanza

	// URIPath
	token := s.ScanUseful()
	if token.typ != STRING || token.val[0] != '/' {
		return h, fmt.Errorf("Parse error in 'handle' stanza: expecting a URI path starting with '/', got %q", token)
	}

	h.URIPath = token.val

	// Begin block
	token = s.ScanUseful()
	if token.typ != OPEN_CURLY {
		return h, fmt.Errorf("Parse error in 'handle' stanza: expecting '{', got %q", token)
	}

	for {
		token = s.ScanUseful()
		if token.typ == CLOSE_CURLY {
			break
		}

		if token.typ == EXPECT {
			exp := Expect{}
			err := exp.Parse(s)
			if err != nil {
				return h, err
			}
			h.Expectations = append(h.Expectations, exp)
		}

		if token.typ == TX {
			h.Response = TxResp{}
			err := h.Response.Parse(s)
			if err != nil {
				return h, err
			}
			// Sending the response is the last allowed action in a 'handle'
			// block
			token = s.ScanUseful()
			if token.typ != CLOSE_CURLY {
				return h, fmt.Errorf("Parse error in 'handle' stanza: expecting '}' after 'tx' command, got %q", token)
			} else {
				// End block
				break
			}
		}
	}

	return h, nil
}

func parseClient(s *scanner) (ClientStanza, error) {
	var c ClientStanza
	var err error

	// Client name
	token := s.ScanUseful()
	if token.typ != STRING {
		return c, fmt.Errorf("Parse error in 'client' stanza: expecting a name for the client, got %q", token)
	}

	c.Name = token.val

	// Begin block
	token = s.ScanUseful()
	if token.typ != OPEN_CURLY {
		return c, fmt.Errorf("Parse error in 'client' stanza: expecting '{', got %q", token)
	}

	for {
		token = s.ScanUseful()
		if token.typ == CLOSE_CURLY {
			break
		}
		if token.typ == TX {
			c.Request = TxReq{}
			err = c.Request.Parse(s)
			if err != nil {
				return c, err
			}
		}
		if token.typ == EXPECT {
			exp := Expect{}
			err := exp.Parse(s)
			if err != nil {
				return c, err
			}
			c.Expectations = append(c.Expectations, exp)
		}
	}
	return c, nil
}

// Parse returns a list of handlers and clients upon successful parsing of the
// given HTC program passed as a io.Reader
func Parse(r io.Reader) ([]HandleStanza, []ClientStanza, error) {
	var h []HandleStanza
	var c []ClientStanza

	s := newScanner(r)

	for {
		token := s.ScanUseful()
		if token.typ == EOF {
			break
		}
		if token.typ == ILLEGAL {
			return h, c, fmt.Errorf("Parse error: %s", token)
		}
		if token.typ == HANDLE {
			hs, err := parseHandle(s)
			if err != nil {
				return h, c, err
			}

			h = append(h, hs)
		}
		if token.typ == CLIENT {
			cs, err := parseClient(s)
			if err != nil {
				return h, c, err
			}

			c = append(c, cs)
		}
	}

	if len(h) == 0 && len(c) == 0 {
		return h, c, fmt.Errorf("Parse error: at least one of 'handle' or 'client' stanza are needed")
	}

	return h, c, nil
}
