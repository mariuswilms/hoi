// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"errors"
	"fmt"
	"hash/adler32"
	"log"
	"path/filepath"
)

type ProjectDirective struct {
	Name                 string
	Path                 string
	Context              string
	UsePHP               bool
	UsePHPLegacyRewrites bool
	PHPVersion           string
	UseLargeUploads      bool
	UseMediaVersions     bool
	UseMediaTransfers    bool
	UseFiles             bool
	UseAssets            bool
	// Whether to use classic img/js/css dirs.
	UseClassicAssets bool
	// host all subpaths with a prefixed
	// undersore i.e. /media under /_media
	UseNoConflict bool
}

func (c *ProjectDirective) ID() string {
	if c.Path == "" {
		log.Fatal(errors.New("no path to generate ID"))
	}
	return fmt.Sprintf("%x", adler32.Checksum([]byte(c.Path)))
}

func (c *ProjectDirective) PrettyName() string {
	if c.Name != "" {
		if c.Context != "" {
			return fmt.Sprintf("%s@%s", c.Name, c.Context)
		}
		return fmt.Sprintf("%s@?", c.Name)
	}
	return fmt.Sprintf("? in %s", filepath.Base(c.Path))
}
