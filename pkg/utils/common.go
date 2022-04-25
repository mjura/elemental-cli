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

package utils

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/distribution/distribution/reference"
	"github.com/joho/godotenv"
	cnst "github.com/rancher-sandbox/elemental/pkg/constants"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/twpayne/go-vfs"
	"github.com/zloylos/grsync"
)

func CommandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// BootedFrom will check if we are booting from the given label
func BootedFrom(runner v1.Runner, label string) bool {
	out, _ := runner.Run("cat", "/proc/cmdline")
	return strings.Contains(string(out), label)
}

// GetDeviceByLabel will try to return the device that matches the given label.
// attempts value sets the number of attempts to find the device, it
// waits a second between attempts.
func GetDeviceByLabel(runner v1.Runner, label string, attempts int) (string, error) {
	part, err := GetFullDeviceByLabel(runner, label, attempts)
	if err != nil {
		return "", err
	}
	return part.Path, nil
}

// GetFullDeviceByLabel works like GetDeviceByLabel, but it will try to get as much info as possible from the existing
// partition and return a v1.Partition object
func GetFullDeviceByLabel(runner v1.Runner, label string, attempts int) (*v1.Partition, error) {
	for tries := 0; tries < attempts; tries++ {
		_, _ = runner.Run("udevadm", "settle")
		parts, err := GetAllPartitions()
		if err != nil {
			return nil, err
		}
		for _, part := range parts {
			if part.Label == label {
				return part, nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return nil, errors.New("no device found")
}

// CopyFile Copies source file to target file using Fs interface. If target
// is  directory source is copied into that directory using source name file.
func CopyFile(fs v1.FS, source string, target string) (err error) {
	if dir, _ := IsDir(fs, target); dir {
		target = filepath.Join(target, filepath.Base(source))
	}
	sourceFile, err := fs.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = sourceFile.Close()
		}
	}()

	targetFile, err := fs.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = targetFile.Close()
		}
	}()

	_, err = io.Copy(targetFile, sourceFile)
	return err
}

// Copies source file to target file using Fs interface
func CreateDirStructure(fs v1.FS, target string) error {
	for _, dir := range []string{"/run", "/dev", "/boot", "/usr/local", "/oem"} {
		err := MkdirAll(fs, filepath.Join(target, dir), cnst.DirPerm)
		if err != nil {
			return err
		}
	}
	for _, dir := range []string{"/proc", "/sys"} {
		err := MkdirAll(fs, filepath.Join(target, dir), cnst.NoWriteDirPerm)
		if err != nil {
			return err
		}
	}
	err := MkdirAll(fs, filepath.Join(target, "/tmp"), cnst.DirPerm)
	if err != nil {
		return err
	}
	// Set /tmp permissions regardless the umask setup
	err = fs.Chmod(filepath.Join(target, "/tmp"), cnst.TempDirPerm)
	if err != nil {
		return err
	}
	return nil
}

// SyncData rsync's source folder contents to a target folder content,
// both are expected to exist before hand.
func SyncData(fs v1.FS, source string, target string, excludes ...string) error {
	if fs != nil {
		if s, err := fs.RawPath(source); err == nil {
			source = s
		}
		if t, err := fs.RawPath(target); err == nil {
			target = t
		}
	}

	if !strings.HasSuffix(source, "/") {
		source = fmt.Sprintf("%s/", source)
	}

	if !strings.HasSuffix(target, "/") {
		target = fmt.Sprintf("%s/", target)
	}

	task := grsync.NewTask(
		source,
		target,
		grsync.RsyncOptions{
			Quiet:   false,
			Archive: true,
			XAttrs:  true,
			ACLs:    true,
			Exclude: excludes,
		},
	)

	err := task.Run()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.Join([]string{task.Log().Stderr, task.Log().Stdout}, "\n"))
	}

	return nil
}

// Reboot reboots the system afater the given delay (in seconds) time passed.
func Reboot(runner v1.Runner, delay time.Duration) error {
	time.Sleep(delay * time.Second)
	_, err := runner.Run("reboot", "-f")
	return err
}

// Shutdown halts the system afater the given delay (in seconds) time passed.
func Shutdown(runner v1.Runner, delay time.Duration) error {
	time.Sleep(delay * time.Second)
	_, err := runner.Run("poweroff", "-f")
	return err
}

// CosignVerify runs a cosign validation for the give image and given public key. If no
// key is provided then it attempts a keyless validation (experimental feature).
func CosignVerify(fs v1.FS, runner v1.Runner, image string, publicKey string, debug bool) (string, error) {
	args := []string{}

	if debug {
		args = append(args, "-d=true")
	}
	if publicKey != "" {
		args = append(args, "-key", publicKey)
	} else {
		os.Setenv("COSIGN_EXPERIMENTAL", "1")
		defer os.Unsetenv("COSIGN_EXPERIMENTAL")
	}
	args = append(args, image)

	// Give each cosign its own tuf dir so it doesnt collide with others accessing the same files at the same time
	tmpDir, err := TempDir(fs, "", "cosign-tuf-")
	if err != nil {
		return "", err
	}
	_ = os.Setenv("TUF_ROOT", tmpDir)
	defer func(fs v1.FS, path string) {
		_ = fs.RemoveAll(path)
	}(fs, tmpDir)
	defer func() {
		_ = os.Unsetenv("TUF_ROOT")
	}()

	out, err := runner.Run("cosign", args...)
	return string(out), err
}

// CreateSquashFS creates a squash file at destination from a source, with options
// TODO: Check validity of source maybe?
func CreateSquashFS(runner v1.Runner, logger v1.Logger, source string, destination string, options []string) error {
	// create args
	args := []string{source, destination}
	// append options passed to args in order to have the correct order
	args = append(args, options...)
	out, err := runner.Run("mksquashfs", args...)
	if err != nil {
		logger.Debugf("Error running squashfs creation, stdout: %s", out)
		logger.Errorf("Error while creating squashfs from %s to %s: %s", source, destination, err)
		return err
	}
	return nil
}

// LoadEnvFile will try to parse the file given and return a map with the kye/values
func LoadEnvFile(fs v1.FS, file string) (map[string]string, error) {
	var envMap map[string]string
	var err error

	f, err := fs.Open(file)
	if err != nil {
		return envMap, err
	}
	defer f.Close()

	envMap, err = godotenv.Parse(f)
	if err != nil {
		return envMap, err
	}

	return envMap, err
}

// GetUpgradeTempDir returns the dir for storing upgrade related temporal files
// It will respect TMPDIR and use that if exists, fallback to try the persistent partition if its mounted
// and finally the default /tmp/ dir
func GetUpgradeTempDir(config *v1.RunConfig) string {
	// if we got a TMPDIR var, respect and use that
	dir := os.Getenv("TMPDIR")
	if dir != "" {
		return filepath.Join(dir, "elemental-upgrade")
	}
	// Check persistent and if its mounted
	persistent, err := GetFullDeviceByLabel(config.Runner, config.PersistentLabel, 5)
	if err == nil && persistent.MountPoint != "" {
		return filepath.Join(persistent.MountPoint, "elemental-upgrade")
	}
	return filepath.Join("/", "tmp", "elemental-upgrade")
}

// IsLocalURI returns true if the uri has "file" scheme or no scheme and URI is
// not prefixed with a domain (container registry style). Returns false otherwise.
// Error is not nil only if the url can't be parsed.
func IsLocalURI(uri string) (bool, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return false, err
	}
	if u.Scheme == "file" {
		return true, nil
	}
	if u.Scheme == "" {
		// Check first part of the path is not a domain (e.g. registry.suse.com/elemental)
		// reference.ParsedNamed expects a <domain>[:<port>]/<path>[:<tag>] form.
		if _, err = reference.ParseNamed(uri); err != nil {
			return true, nil
		}
	}
	return false, nil
}

// IsHTTPURI returns true if the uri has "http" or "https" scheme, returns false otherwise.
// Error is not nil only if the url can't be parsed.
func IsHTTPURI(uri string) (bool, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return false, err
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		return true, nil
	}
	return false, nil
}

// GetSource copies given source to destination, if source is a local path it simply
// copies files, if source is a remote URL it tries to download URL to destination.
func GetSource(config *v1.RunConfig, source string, destination string) error {
	local, err := IsLocalURI(source)
	if err != nil {
		config.Logger.Errorf("Not a valid url: %s", source)
		return err
	}

	err = vfs.MkdirAll(config.Fs, filepath.Dir(destination), cnst.DirPerm)
	if err != nil {
		return err
	}
	if local {
		u, _ := url.Parse(source)
		err = CopyFile(config.Fs, u.Path, destination)
		if err != nil {
			return err
		}
	} else {
		err = config.Client.GetURL(config.Logger, source, destination)
		if err != nil {
			return err
		}
	}
	return nil
}

// ValidContainerReferece returns true if the given string matches
// a container registry reference, false otherwise
func ValidContainerReference(ref string) bool {
	if _, err := reference.ParseNormalizedNamed(ref); err != nil {
		return false
	}
	return true
}

// ValidTaggedContainerReferece returns true if the given string matches
// a container registry reference including a tag, false otherwise.
func ValidTaggedContainerReference(ref string) bool {
	n, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return false
	}
	if reference.IsNameOnly(n) {
		return false
	}
	return true
}

// NewSrcGuessingType returns new v1.ImageSource instance guessing its type
// applying somne heuristic techniques (by order of preference):
//   1. Assume it is Dir/File if value is found as a path in host
//	 2. Assume it is a container registry reference if it matches [<domain>/]<repositry>:<tag>
//      (only domain is optional)
//	 3. Fallback to a channel source
func NewSrcGuessingType(c v1.Config, value string) v1.ImageSource {
	if exists, _ := Exists(c.Fs, value); exists {
		if dir, _ := IsDir(c.Fs, value); dir {
			return v1.NewDirSrc(value)
		}
		return v1.NewFileSrc(value)
	} else if ValidTaggedContainerReference(value) {
		return v1.NewDockerSrc(value)
	}
	return v1.NewChannelSrc(value)
}
