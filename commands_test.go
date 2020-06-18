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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpectToString(t *testing.T) {
	exp := Expect{
		verbatim: "status eq 200",
	}
	assert.Equal(t, "\"status eq 200\"", exp.String())
}

func TestExpectParse(t *testing.T) {
	s := newScanner(strings.NewReader("req.method eq \"GET\""))
	exp := Expect{}
	err := exp.Parse(s)
	assert.Equal(t, err, nil)
	assert.Equal(t, "GET", exp.expected)
}

func TestExpectRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	exp := Expect{field: EXPECT_METHOD, operator: EQUAL, expected: "GET"}

	assert.Equal(t, true, exp.Request(*req))
}

func TestExpectRequestPanic(t *testing.T) {
	var exp Expect
	var r http.Request

	// Regular expression that fails to compile
	exp = Expect{
		field:    EXPECT_METHOD,
		operator: TILDE,
		expected: "(invalid-regular-expression",
	}
	assert.Panics(t, func() { exp.Request(r) })

	// Invalid operator (42)
	exp = Expect{
		field:    EXPECT_METHOD,
		operator: 42,
		expected: "",
	}

	assert.Panics(t, func() { exp.Request(r) })
}

func TestTxRespToString(t *testing.T) {
	r := TxResp{
		statusCode: 404,
	}
	assert.Equal(t, "tx HTTP 404: \"\"", r.String())
}

func TestTxRespSend(t *testing.T) {
	w := httptest.NewRecorder()

	h := map[string]string{
		"X-Served-By": "httptester",
		"X-Hello":     "world",
	}
	r := TxResp{
		statusCode: 200,
		headers:    h,
		body:       "Hello world!",
	}

	assert.True(t, r.Send(w))

	resp := w.Result()

	assert.Equal(t, r.statusCode, resp.StatusCode)

	// Check that all headers are returned
	for key, value := range h {
		assert.Equal(t, value, resp.Header.Get(key))
	}

	// Check the response body
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, r.body, string(body))
}
