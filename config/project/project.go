// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package project

import (
	"errors"
	"fmt"
	"hash/adler32"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
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

func (c *ProjectDirective) Augment() error {
	log.Printf("discovering project config: %s", c.Path)

	if _, err := os.Stat(c.Path + "/app/webroot/index.php"); err == nil {
		log.Print("- using PHP")
		c.UsePHP = true

		legacy, err := fileContainsString(c.Path+"/app/webroot/index.php", "cake")
		if err != nil {
			return err
		}
		if legacy {
			log.Print("- using legacy rewrites")
			c.UsePHPLegacyRewrites = true
		}
		log.Print("- using large uploads")
		c.UseLargeUploads = true
	}

	if _, err := os.Stat(c.Path + "/assets"); err == nil {
		c.UseAssets = true
	}
	if _, err := os.Stat(c.Path + "/media/versions"); err == nil {
		c.UseMediaVersions = true
	}
	if _, err := os.Stat(c.Path + "/media/transfers"); err == nil {
		c.UseMediaTransfers = true
	}
	if _, err := os.Stat(c.Path + "/files"); err == nil {
		c.UseFiles = true
	}
	if _, err := os.Stat(c.Path + "/app/webroot/css"); err == nil {
		c.UseAssets = true
		c.UseClassicAssets = true
	}
	return nil
}

func fileContainsString(file string, search string) (bool, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return false, err
	}
	s := string(b)
	return strings.Contains(s, search), nil
}
