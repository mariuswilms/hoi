// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func generateProjectConfig(sPath string, tPath string, tmplData interface{}) error {
	if _, err := os.Stat(sPath); os.IsNotExist(err) {
		return err
	}
	exchanger := strings.NewReplacer(sPath, tPath) // exchange bases
	dirs := make(map[string]string)
	files := make(map[string]string)
	templates := make(map[string]*template.Template) // Keyed by target path.

	err := filepath.Walk(sPath, func(path string, f os.FileInfo, err error) error {
		xPath := exchanger.Replace(path)

		if f.IsDir() {
			dirs[xPath] = path
		} else {
			files[xPath] = path
		}
		return nil
	})
	if err != nil {
		return err
	}

	for dst, src := range files {
		t, err := loadTemplate(src)
		if err != nil {
			return err
		}
		templates[dst] = t
	}

	for dst, _ := range dirs {
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			continue // do not create already existing
		}
		log.Printf("creating directory: %s", prettyPath(dst))

		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}
	}
	for dst, t := range templates {
		if err := writeTemplate(t, dst, tmplData); err != nil {
			return err
		}
	}
	return nil
}

func loadTemplate(path string) (*template.Template, error) {
	log.Printf("loading template: %s", prettyPath(path))

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t := template.New(prettyPath(path)) // use this as identifier
	return t.Parse(string(bytes))
}

// FIXME clean up partially written file
func writeTemplate(t *template.Template, dst string, tmplData interface{}) error {
	log.Printf("writing file: %s", prettyPath(dst))

	fh, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, 0640)
	if err != nil {
		return err
	}
	defer fh.Close()
	return t.Execute(fh, tmplData)
}

func prettyPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	return strings.Replace(path, cwd, ".", 1)
}
