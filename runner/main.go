// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Runners manage configurations in files and services to
// fullfill the needs of the project. They usually command
// a builder and a utilize a system, into which artifacts are
// installed.
package runner

// Runnable describes methods common to each runner. Runners often use a builder
// and a system (into which built configuration files are installed).
//
// Runners are "dumb" in that they do not track which configuration depends on
// which configuration and needs to be rebuilt. Instead it simply alwas does a
// full rebuild. The order in which Runnable methods should be invoked for a
// full rebuild is:
//
//   Disable -> Clean -> Build -> Enable -> Commit
//
// As these methods are invoked in a sequential way they are called "steps."
// Method signature is intentionally kept simple and equal as we want to tread
// these methods as the abstract kind step.
type Runnable interface {
	// Builds configuration files.
	Build() error
	// Removes any build files that had been created with Build().
	Clean() error
	// Installs configuration files into runner's system and
	// activates them there.
	Enable() error
	// Removes configuration files from runner's system and
	// deactivates them.
	Disable() error
	// Commits any changes made to the system.
	Commit() error
}
