// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"bytes"
	"io"
	"os"
	"strings"
	"text/template"
)

func CopyFile(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, info.Mode())
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	return d.Sync()
}

// Parses and executes a template in one step. Will only parse if necessary.
func ParseAndExecuteTemplate(name string, tmpl string, data interface{}) (string, error) {
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}
	t := template.New(name)

	if _, err := t.Parse(tmpl); err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)

	if err := t.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
