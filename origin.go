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
	"log"
	"net/http"
)

type Origin struct {
	errors  []error
	port    int
	verbose bool
}

func NewOrigin(port int, verbose bool) Origin {
	return Origin{port: port, verbose: verbose}
}

func (o *Origin) addHandler(hs HandleStanza) {
	http.HandleFunc(hs.URIPath, func(w http.ResponseWriter, req *http.Request) {
		// Expect things
		for _, exp := range hs.Expectations {
			if o.verbose {
				log.Println("Expecting", exp)
			}
			if exp.Request(*req) == false {
				o.errors = append(o.errors, fmt.Errorf("FAILED: %s (actual=%q)", exp, exp.ActualRequest(*req)))
			}
		}

		// return response
		hs.Response.Send(w)
	})
}

func (o Origin) start() {
	http.HandleFunc("/httpTesterInternalCheck", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "UP!")
	})
	go http.ListenAndServe(fmt.Sprintf(":%d", o.port), nil)

	waitForGET(fmt.Sprintf("http://localhost:%d/httpTesterInternalCheck", o.port))
}
