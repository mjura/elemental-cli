/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package http_test

import (
	"os"
	"path/filepath"

	"github.com/rancher-sandbox/elemental/pkg/http"
	"github.com/rancher-sandbox/elemental/pkg/types/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const source = "https://github.com/rancher-sandbox/elemental/releases/download/v0.0.13/elemental-v0.0.13-Linux-x86_64.tar.gz"

var _ = Describe("HTTPClient", Label("http"), func() {
	var client *http.Client
	var log v1.Logger
	var destDir string
	BeforeEach(func() {
		client = http.NewClient()
		log = v1.NewNullLogger()
		destDir, _ = os.MkdirTemp("", "elemental-test")
	})
	AfterEach(func() {
		os.RemoveAll(destDir)
	})
	It("Downloads a test file to destination folder", func() {
		// Download a public elemental release
		_, err := os.Stat(filepath.Join(destDir, "elemental-v0.0.13-Linux-x86_64.tar.gz"))
		Expect(err).NotTo(BeNil())
		Expect(client.GetURL(log, source, destDir)).To(BeNil())
		_, err = os.Stat(filepath.Join(destDir, "elemental-v0.0.13-Linux-x86_64.tar.gz"))
		Expect(err).To(BeNil())
	})
	It("Downloads a test file to some specified destination file", func() {
		// Download a public elemental release
		_, err := os.Stat(filepath.Join(destDir, "testfile"))
		Expect(err).NotTo(BeNil())
		Expect(client.GetURL(log, source, filepath.Join(destDir, "testfile"))).To(BeNil())
		_, err = os.Stat(filepath.Join(destDir, "testfile"))
		Expect(err).To(BeNil())
	})
	It("Fails to download a non existing url", func() {
		source := "http://nonexisting.stuff"
		Expect(client.GetURL(log, source, destDir)).NotTo(BeNil())
	})
})
