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

// The goal of httptester is to ease testing of HTTP proxies. Write tests in
// the HTC language (Httptester Test Case) and run them against the proxy of
// your choice. No dependencies needed except for the proxy server itself (ATS
// only for now).
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

var verbose = flag.Bool("verbose", false, "enable verbose mode")

func waitForGET(url string) {
	for {
		time.Sleep(200 * time.Millisecond)

		resp, err := http.Get(url)
		if err == nil {
			if resp.StatusCode != 200 {
				log.Fatalf("Unexpected status code received from url %s: %d\n", url, resp.StatusCode)
			} else {
				break
			}
		}
	}

	if *verbose {
		log.Println("Finished waiting for", url)
	}
}

func freePortOrDie() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] file\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	// Start origin server and proxy
	originPort := freePortOrDie()
	proxyPort := freePortOrDie()

	origin := NewOrigin(originPort)
	origin.start()

	proxy := NewProxy(proxyPort, originPort)
	proxy.start()
	if *verbose {
		log.Println("Proxy started using temporary directory", proxy.tmpDir)
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	h, c, err := Parse(f)
	if err != nil {
		log.Fatal(err)
	}

	// Iterate over HandleStanzas
	for _, hs := range h {
		origin.addHandler(hs)
	}

	// Start clients
	for _, cs := range c {
		addr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
		resp, err := cs.Request.Send(addr)
		if *verbose {
			log.Println("Sending", cs.Request)
		}

		if err != nil {
			log.Fatal(err)
		}

		for _, exp := range cs.Expectations {
			if exp.Response(*resp) == false {
				proxy.stop()
				log.Println(cs.Request)
				log.Println(exp.StringResponse(*resp))
				log.Fatalf("FAILED: %s (actual=%q)", exp, exp.ActualResponse(*resp))
			}
		}
	}

	proxy.stop()

	if len(origin.errors) > 0 {
		log.Fatal(origin.errors[0])
	}

	// Remove temporary directory only if tests passed
	proxy.cleanup()

	os.Exit(0)
}
