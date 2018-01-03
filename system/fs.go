// Copyright 2017 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"archive/tar"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

func NewFilesystem(p *project.Config, s *server.Config) *Filesystem {
	return &Filesystem{p: p, s: s}
}

// Handles operations within the local filesystem.
type Filesystem struct {
	p *project.Config
	s *server.Config
}

// Sets up the source end of the volume, the target is automatically
// created by the bind mount. This setup is intentionally kept simply
// and does not try to protect files inside the project, more than
// using standard permission settings.
func (sys Filesystem) SetupVolume(v project.VolumeDirective) error {
	runPath := v.GetRunPath(sys.p, sys.s)
	src := v.GetSource(sys.p, sys.s)

	// Create the project directory in each with restrictive
	// permissions. We assume hoi is running under root, so there is
	// no need to further restrict ownership.
	if _, err := os.Stat(runPath); os.IsNotExist(err) {
		if err := os.MkdirAll(runPath, 0700); err != nil {
			return err
		}
	}

	// Contained actual source directories may then use other
	// permissions. They are bind mounted and tree traversal isn't
	// necessary to see their contents.
	if _, err := os.Stat(src); os.IsNotExist(err) {
		// Mkdir honors system's umask, to get around backing up then
		// resetting umask, we chmod afterwards.
		if err := os.Mkdir(src, 0700); err != nil {
			return err
		}

		if err := os.Chmod(src, 0755); err != nil {
			return err
		}

		// Use our own (poor-man's) Chown here, so we do not need to
		// lookup the uid/gid, which would require cgo, which isn't
		// available during cross compilation.
		if err := exec.Command("chown", sys.s.User+":"+sys.s.Group, src).Run(); err != nil {
			return err
		}
	}
	return nil
}

// Intentionally not compressing data as we can assume it is mostly
// pre-compressed media data.
func (sys Filesystem) DumpVolume(v project.VolumeDirective, tw *tar.Writer) error {
	source := v.GetSource(sys.p, sys.s)
	base := filepath.Base(source)

	return filepath.Walk(source, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(f, path)
		if err != nil {
			return err
		}

		if base != "" {
			header.Name = filepath.Join(base, strings.TrimPrefix(path, source))
		}
		if f.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Include header for directories and symlinks but not the
		// actual contents (there is none).
		if !f.Mode().IsRegular() {
			return nil
		}

		fr, err := os.Open(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(tw, fr)

		fr.Close()
		return err
	})
}
