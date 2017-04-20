// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Augments a project configuration as read from a Hoifile, so that
// most configuration does not have to be given explictly and project
// configuration can stay lean.
func (cfg *Config) Augment() error {
	log.Printf("discovering project config: %s", cfg.Path)

	// Volumes might not yet be mounted, still we want to serve
	// data from them. On the other hand directories might simply
	// exists without being placed on a volume.
	hasDirectory := func(path string) bool {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		for _, volume := range cfg.Volume {
			if volume.GetTarget(cfg) == path {
				return true
			}
		}
		return false
	}

	if cfg.Name == "" {
		// Strips the directory name from known context suffix, the
		// context may be added as suffixed later (see database name).
		cfg.Name = strings.TrimSuffix(filepath.Base(cfg.Path), fmt.Sprintf("_%s", cfg.Context))
		log.Printf("- guessed project name: %s", cfg.Name)
	}

	// Discover the webroot by looking a common names and files
	// contained within such a directory. We must take care to not
	// mistakenly expose a directory publicly with contains sensitive
	// material.
	//
	// If we find a directory named "webroot" this is a strong
	// indication it is intended as such.
	//
	// When not finding any directory with this name we'll start
	// looking into the root directory for index.php or index.html
	// files in order to confirm root is the webroot.
	//
	// No other directories except they are named "webroot" or the
	// root directory can become webroot.
	var breakWalk = errors.New("stopped walk early")

	// For performance reasons look in common places first, than
	// fallback to walking the entire tree.
	if _, err := os.Stat(cfg.Path + "/app/webroot"); err == nil {
		cfg.Webroot = "app/webroot"
	} else {
		err := filepath.Walk(cfg.Path, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !f.IsDir() {
				return filepath.SkipDir
			}
			if f.Name() != "webroot" {
				return filepath.SkipDir
			}
			cfg.Webroot = path
			return breakWalk
		})
		if err != nil && err != breakWalk {
			return fmt.Errorf("failed to detect webroot: %s", err)
		}

		if cfg.Webroot == "" {
			_, errPHP := os.Stat(cfg.Path + "/index.php")
			_, errHTML := os.Stat(cfg.Path + "/index.html")
			if errPHP == nil || errHTML == nil {
				cfg.Webroot = "."
			}
		}
	}
	if cfg.Webroot == "" {
		return fmt.Errorf("failed to detect webroot in: %s", cfg.Path)
	} else {
		log.Printf("- found webroot in: %s", cfg.Webroot)
	}

	// Detect which type of application this is first.
	if _, err := os.Stat(cfg.GetAbsoluteWebroot() + "/index.html"); err == nil {
		log.Print("- detected static project")
		cfg.Kind = KindStatic
	} else {
		if _, err := os.Stat(cfg.GetAbsoluteWebroot() + "/index.php"); err == nil {
			log.Print("- detected PHP project")
			cfg.Kind = KindPHP
		} else if _, err := os.Stat(cfg.Path + "/app/composer.json"); err == nil {
			log.Print("- detected PHP project")
			cfg.Kind = KindPHP
		} else {
			return fmt.Errorf("failed to detect project type in: %s", cfg.Path)
		}
	}

	log.Print("- found front controller, routing requests through it")
	cfg.UseFrontController = true

	if cfg.Kind == KindPHP && cfg.UseFrontController {
		// Detect oldish versions of CakePHP by inspecting the front controller
		// file for certain string patterns. CakePHP version >= use uppercased "Cake"
		// string.
		legacy, err := fileContainsString(cfg.GetAbsoluteWebroot()+"/index.php", "cake")
		if err != nil {
			return err
		}
		if legacy {
			log.Print("- using legacy front controller")
			cfg.UseLegacyFrontController = true
		}
	}

	// /assets and /media_versions can be either in the root
	// of the project or nested under webroot, let's check
	// if the are nested or not by looking at /assets.

	if hasDirectory(cfg.GetAbsoluteWebroot() + "/css") {
		log.Print("- using classic assets directories ('css'/'img'/'js')")
		cfg.UseAssets = true
		cfg.UseClassicAssets = true
	}

	if cfg.UseClassicAssets || hasDirectory(cfg.GetAbsoluteWebroot()+"/assets") {
		log.Print("- using webroot nesting")
		cfg.UseWebrootNesting = true
	}
	hasDirectoryInRoot := func(dir string) bool {
		var base string

		if cfg.UseWebrootNesting {
			base = cfg.GetAbsoluteWebroot()
		} else {
			base = cfg.Path
		}
		return hasDirectory(filepath.Join(base, dir))
	}

	if hasDirectoryInRoot("assets") {
		log.Print("- serving unified assets directory ('assets')")
		cfg.UseAssets = true
	}

	if hasDirectoryInRoot("media_versions") {
		log.Print("- serving media versions ('media_versions')")
		cfg.UseMediaVersions = true
	}
	if hasDirectoryInRoot("media") {
		log.Print("- serving media transfers ('media') internally")
		cfg.UseMediaTransfers = true
	}

	if hasDirectoryInRoot("files") {
		log.Print("- serving files ('files') internally")
		cfg.UseFiles = true
	}

	if cfg.UseMediaTransfers {
		log.Print("- enabling uploads")
		cfg.UseUploads = true
	}

	// Guesses auth user names. An empty user name usually indicates
	// that auth is disabled. However, here we interpret non empty
	// passwords as an indicator for enabled auth. This will than
	// trigger the correct behavior in GetCreds().
	for k, _ := range cfg.Domain {
		e := cfg.Domain[k]

		if e.Auth.Password != "" && e.Auth.User == "" {
			e.Auth.User = cfg.Name
			log.Printf("- guessed auth user: %s", e.Auth.User)
		}
		cfg.Domain[k] = e
	}

	// Guessing will always give the same result, we can therefore
	// only guess once.
	guessedDBName := false
	for k, _ := range cfg.Database {
		e := cfg.Database[k]
		if e.Name == "" {
			if guessedDBName {
				return fmt.Errorf("more than one database name to guess; giving up on augmenting: %s", cfg.Path)
			}
			// Production and local development databases are not
			// suffixed with context name. For other contexts the
			// database name will look like "example_stage".
			if cfg.Context == ContextProduction || cfg.Context == ContextDevelopment {
				e.Name = cfg.Name
			} else {
				e.Name = fmt.Sprintf("%s_%s", cfg.Name, cfg.Context)
			}
			log.Printf("- guessed database name: %s", e.Name)
			guessedDBName = true
		}
		if e.User == "" {
			// User name corresponds to database name and follows the
			// same suffixing rules as the database name.
			e.User = e.Name
			log.Printf("- guessed database user: %s", e.User)
		}
		cfg.Database[k] = e
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
