// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Runners manage configurations in files and services to fullfill the
// needs of the project. They usually command a builder and utilize
// one or multiple systems, into which artifacts are installed.
package runner

// Runnable describes methods common to each runner. Runnable methods are called
// "steps" as these methods are invoked one after another in a fixed order. Steps
// do not take any arguments as to being able to treat them equally.
//
// When configuration changes, runners make no effort to determine if
// a rebuild is required. They simply always rebuild:
//
//   Disable -> Enable -> Commit
type Runnable interface {
	// Builds and installs configuration files into runner's system and
	// activates them there.
	Enable() error
	// Removes configuration files from runner's system and
	// deactivates them. Removes any build files.
	Disable() error
	// Commits any changes made to the system.
	Commit() error
}
