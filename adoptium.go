// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mholt/archiver/v3"
)

// Mirrors the Adoptium "Release" schema
type adoptiumRelease struct {
	Binaries []adoptiumBinary `json:"binaries"`
	Binary   adoptiumBinary   `json:"binary"`
}

// Mirrors the Adoptium "Binary" schema
type adoptiumBinary struct {
	Architecture   string          `json:"architecture" binding:"required"`
	Implementation string          `json:"jvm_impl" binding:"required"`
	Platform       string          `json:"os" binding:"required"`
	Package        adoptiumPackage `json:"package" binding:"required"`
}

// Mirrors the Adoptium "Package" schema
type adoptiumPackage struct {
	Name string `json:"name" binding:"required"`
	Link string `json:"link" binding:"required"`
}

// Only one runtime can be downloaded at a time. This is to prevent issues with
// partial downloads.
var downloadLock sync.Mutex

// LookupRelease finds a release for the given attributes.
func lookupRelease(arch, platform, implementation, version string) (*adoptiumBinary, error) {

	url := fmt.Sprintf("https://api.adoptopenjdk.net/v3/assets/version/%s?jvm_impl=%s&os=%s&architecture=%s", version, implementation, platform, arch)
	log.Println("GET:", url)
	res, err := adoptium.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var releases []adoptiumRelease
	if err := json.NewDecoder(res.Body).Decode(&releases); err != nil {
		return nil, err
	}

	if len(releases) > 0 && len(releases[0].Binaries) > 0 {
		return &releases[0].Binaries[0], nil
	}

	return nil, errors.New("No release found")
}

// LookupLatestRelease finds the latest release for the given attributes.
func lookupLatestRelease(arch, platform, implementation string, majorVersion int) (*adoptiumBinary, error) {

	url := fmt.Sprintf("https://api.adoptopenjdk.net/v3/assets/latest/%d/%s", majorVersion, implementation)
	log.Println("GET:", url)
	res, err := adoptium.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var releases []adoptiumRelease
	if err := json.NewDecoder(res.Body).Decode(&releases); err != nil {
		return nil, err
	}

	for _, release := range releases {
		if release.Binary.Platform == platform && release.Binary.Architecture == arch {
			return &release.Binary, nil
		}
	}

	return nil, errors.New("No release found")
}

// DownloadRelease downloads a runtime image to the cache directory and returns
// the path to the extracted runtime directory.
func downloadRelease(binary *adoptiumBinary) (string, error) {
	downloadLock.Lock()
	defer downloadLock.Unlock()

	runtimePath := RT_CACHE + "/" + strings.TrimSuffix(strings.TrimSuffix(binary.Package.Name, ".zip"), ".tar.gz")

	// Check if the runtime is cached first
	if _, e := os.Stat(runtimePath); !os.IsNotExist(e) {
		contents, _ := filepath.Glob(runtimePath + "/*")
		if len(contents) == 1 {
			return contents[0], nil
		} else {
			return "", errors.New("Unexpected runtime contents")
		}
	}

	archivePath, dir := newTemporaryFile(binary.Package.Name)
	defer os.RemoveAll(dir)

	// Download the runtime
	response, err := adoptium.Get(binary.Package.Link)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", errors.New("Abnormal HTTP status code: " + response.Status)
	}
	defer response.Body.Close()

	out, err := os.Create(archivePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		return "", err
	}

	// Extract to the cache directory
	if err := archiver.Unarchive(archivePath, runtimePath); err != nil {
		return "", err
	}

	contents, _ := filepath.Glob(runtimePath + "/*")
	if len(contents) == 1 {
		return contents[0], nil
	} else {
		return "", errors.New("Unexpected runtime contents")
	}
}
