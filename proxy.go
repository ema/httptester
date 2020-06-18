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
	"os"
	"os/exec"
	"path"
)

type Proxy struct {
	port       int
	originPort int
	cmd        *exec.Cmd
	tmpDir     string
}

func NewProxy(port, originPort int) Proxy {
	return Proxy{port: port, originPort: originPort}
}

func writeStringToFile(s string, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	file.WriteString(s)
}

func (p *Proxy) start() {
	// Create temporary directory
	dir, err := ioutil.TempDir("/tmp", "runroot")
	if err != nil {
		log.Fatal(err)
	}
	p.tmpDir = dir

	varDir := path.Join(dir, "var")
	cacheDir := path.Join(varDir, "cache")

	// Create layout file inside the temporary directory
	fname := path.Join(dir, "atslayout.yaml")
	t := `prefix: %s
exec_prefix: %s
bindir: %s/bin
sbindir: %s/sbin
sysconfdir: %s/etc
datadir: %s
includedir: %s/include
libdir: %s/lib
libexecdir: %s/libexec
localstatedir: %s/var
runtimedir: %s/var/run
logdir: %s/var/log
cachedir: %s`
	writeStringToFile(fmt.Sprintf(t, dir, dir, dir, dir, dir, cacheDir, dir, dir, dir, dir, dir, dir, cacheDir), fname)

	// Create ATS layout directory
	cmd := exec.Command("traffic_layout", "init", "-f", "-p", dir, "-l", fname, "--copy-style=soft")

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Create remap.config
	writeStringToFile(fmt.Sprintf("map / http://localhost:%d\n", p.originPort), path.Join(dir, "etc", "remap.config"))

	// Create plugin.config
	writeStringToFile(fmt.Sprintf("xdebug.so\n"), path.Join(dir, "etc", "plugin.config"))

	// Create storage.config
	writeStringToFile(fmt.Sprintf("%s/ 1M\n", cacheDir), path.Join(dir, "etc", "storage.config"))

	// Create records.config
	writeStringToFile(fmt.Sprintf(`CONFIG proxy.config.http.server_ports STRING %d %d:ipv6
#CONFIG proxy.config.http.wait_for_cache INT 2
CONFIG proxy.config.diags.debug.enabled INT 1
`, p.port, p.port), path.Join(dir, "etc", "records.config"))

	// Create ip_allow.config
	writeStringToFile("src_ip=127.0.0.1 action=ip_allow method=ALL\nsrc_ip=::1 action=ip_allow method=ALL\n", path.Join(dir, "etc", "ip_allow.config"))

	// Start traffic_manager
	trafficManager := path.Join(dir, "bin", "traffic_manager")
	p.cmd = exec.Command(trafficManager, "--run-root="+path.Join(dir, "runroot.yaml"))

	err = p.cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	waitForGET(fmt.Sprintf("http://localhost:%d/httpTesterInternalCheck", p.port))
}

func (p Proxy) cleanup() {
	os.RemoveAll(p.tmpDir)
}

func (p Proxy) stop() {
	// Done, shoot ATS
	err := p.cmd.Process.Kill()
	if err != nil {
		log.Println(err)
	}
}
