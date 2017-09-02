// Copyright 2017 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
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
// Command strings may use template syntax (project configuration
// is made available as P). Will parse only when necessarry, most
// commands will not use templating.
//
// When used inside systemd service unit files paths need to be
// absolute. When a command string is non absolute it will be treated
// as being relative to the project root directory and made absolute.
func (c Command) GetCommand(p *Config) (string, error) {
	var cmd string

	if !strings.Contains(c.Command, "{{") {
		cmd = c.Command
	} else {
		cmdTmplData := struct {
			P *Config
		}{
			P: p,
		}
		buf := new(bytes.Buffer)
		cmdT := template.New("cmd")
		cmdT.Parse(c.Command)

		if err := cmdT.Execute(buf, cmdTmplData); err != nil {
			return "", err
		}
		cmd = buf.String()
	}
	if !filepath.IsAbs(cmd) {
		cmd = filepath.Join(p.Path, cmd)
	}
	return cmd, nil
}
