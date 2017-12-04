// Copyright 2017 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"path/filepath"

	"github.com/atelierdisko/hoi/util"
)

type Command struct {
	Command string
}

// Checks whether a command has been provided. Eases access to Command
// property when embededded.
func (c Command) HasCommand() bool {
	return c.Command != ""
}

func (c Command) String() string {
	return c.Command
}

// Returns (parsed and) absolute command string.
//
// When used inside systemd service unit files paths need to be
// absolute. When a command string is non absolute it will be treated
// as being relative to the project root directory and made absolute.
//
// You may use template syntax here (P is the project configuration).
func (c Command) GetCommand(p *Config) (string, error) {
	cmd, err := util.ParseAndExecuteTemplate("command", c.Command, struct {
		P *Config
	}{
		P: p,
	})
	if err != nil {
		return cmd, err
	}
	if !filepath.IsAbs(cmd) {
		cmd = filepath.Join(p.Path, cmd)
	}
	return cmd, nil
}
