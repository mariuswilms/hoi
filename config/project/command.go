// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package project

import (
	"bytes"
	"html/template"
	"log"
	"strings"
)

// Holds a command string which Can be either a path (relative to project root
// or absolute) or a template which evaluates to one of both. Templates may
// reference P (the project configuration).
//
// Commands will be executed with the project root path as the current working
// directory.
type Commando struct {
	Command string
}

func (c *Commando) GetCommand(p Config) (string, error) {
	if !strings.Contains(c.Command, "{{") {
		return c.Command, nil
	}
	log.Printf("parsing command template: %s", c.Command)

	cmdTmplData := struct {
		P Config
	}{
		P: p,
	}
	buf := new(bytes.Buffer)
	cmdT := template.New("cmd")
	cmdT.Parse(c.Command)

	if err := cmdT.Execute(buf, cmdTmplData); err != nil {
		return "", err
	}
	return buf.String(), nil
}
