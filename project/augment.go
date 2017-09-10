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

	webroot, err = cfg.discoverWebroot()
	if err != nil {
		return err
	}
	log.Printf("- found webroot in: %s", webroot)
	cfg.Webroot = webroot

	if cfg.App.Kind == AppKindUnknown {
		if cfg.App.HasCommand() {
			log.Print("- detected service app")
			cfg.App.Kind = AppKindService
		} else if _, err := os.Stat(cfg.GetAbsoluteWebroot() + "/index.html"); err == nil {
			log.Print("- detected static app")
			cfg.App.Kind = AppKindStatic
		} else if _, err := os.Stat(cfg.GetAbsoluteWebroot() + "/index.php"); err == nil {
			log.Print("- detected PHP app")
			cfg.App.Kind = AppKindPHP
		} else if _, err := os.Stat(cfg.Path + "/app/composer.json"); err == nil {
			log.Print("- detected PHP app")
			cfg.App.Kind = AppKindPHP
		} else {
			return fmt.Errorf("failed to detect app kind in: %s", cfg.Path)
		}
	}

	if cfg.App.Kind == AppKindService {
		if cfg.App.Host == "" {
			cfg.App.Host = "localhost"
		}
		if cfg.App.Port == 0 {
			freeport, err := cfg.App.GetFreePort(cfg)
			if err != nil {
				return err
			}
			cfg.App.Port = freeport
			log.Printf("- assigned port %d to app service", cfg.App.Port)
		}
	}

	if cfg.App.Kind == AppKindPHP || cfg.App.Kind == AppKindStatic {
		log.Print("- enabling front controller, routing requests through it")
		cfg.App.UseFrontController = true
	}

	if cfg.App.Kind == AppKindPHP && cfg.App.UseFrontController {
		// Detect oldish versions of CakePHP by inspecting the front controller
		// file for certain string patterns. CakePHP version >= use uppercased "Cake"
		// string.
		legacy, err := fileContainsString(cfg.GetAbsoluteWebroot()+"/index.php", "cake")
		if err != nil {
			return err
		}
		if legacy {
			log.Print("- using legacy front controller")
			cfg.App.UseLegacyFrontController = true
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

// Discover the webroot by looking at common names and files
// contained within such a directory. We must take care not to
// publicly expose a directory that contains sensitive
// material by mistake.
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
func (cfg Config) discoverWebroot() (string, error) {
	// For performance reasons look in common places first, than
	// fallback to walking the entire tree.
	if _, err := os.Stat(cfg.Path + "/app/webroot"); err == nil {
		return "app/webroot", nil
	}

	var breakWalk = errors.New("stopped walk early")
	var string webroot

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
		webroot = path
		return breakWalk
	})
	if err != nil && err != breakWalk {
		return webroot, fmt.Errorf("failed to detect webroot in %s: %s", cfg.Path, err)
	}

	if webroot == "" {
		// Check if webroot is same as root path. Be careful not to expose
		// whole application.
		_, errPHP := os.Stat(cfg.Path + "/index.php")
		_, errHTML := os.Stat(cfg.Path + "/index.html")

		if errPHP == nil || errHTML == nil {
			return ".", nil
		}
		return webroot, fmt.Errorf("failed to detect webroot in %s: %s", cfg.Path, err)
	}
	return webroot, nil
}
