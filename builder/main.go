// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builder

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

const (
	KindWeb    = "web"
	KindPHP    = "php"
	KindCron   = "cron"
	KindWorker = "worker"
)

func NewBuilder(kind string, p project.Config, s server.Config) *Builder {
	return &Builder{kind: kind, p: p, s: s}
}
func NewScopedBuilder(kind string, scope string, p project.Config, s server.Config) *Builder {
	return &Builder{kind: kind, scope: scope, p: p, s: s}
}

// Builds configuration supporting runners.
type Builder struct {
	kind  string
	scope string
	s     server.Config
	p     project.Config
}

func (b Builder) Path() string {
	return filepath.Join(b.s.BuildPath, b.kind, b.p.ID)
}

func (b Builder) ListAvailable() ([]string, error) {
	path := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)
	files := make([]string, 0)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return files, nil
	}
	if b.scope != "" {
		return filepath.Glob(path + "/" + b.scope)
	}
	return filepath.Glob(path + "/*")
}

func (b Builder) Clean() error {
	dir := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)
	log.Printf("cleaning build directory for project %s: %s", b.p.PrettyName(), dir)

	return os.RemoveAll(dir)
}

func (b Builder) WriteFile(name string, reader io.Reader) error {
	log.Printf("writing file for project %s: %s", b.p.PrettyName(), name)

	dir := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	writer, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	return err
}

func (b Builder) LoadTemplate(name string) (*template.Template, error) {
	log.Printf("loading template for project %s: %s", b.p.PrettyName(), name)
	return loadTemplate(filepath.Join(b.s.TemplatePath, b.kind, name))
}

func (b Builder) WriteTemplate(name string, t *template.Template, tmplData interface{}) error {
	log.Printf("compling template for project %s: %s", b.p.PrettyName(), name)

	dir := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return writeTemplate(t, filepath.Join(dir, name), 0644, tmplData)
}

func (b Builder) WriteSensitiveTemplate(name string, t *template.Template, tmplData interface{}) error {
	log.Printf("compiling sensitive template for project %s: %s", b.p.PrettyName(), name)

	dir := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
	}
	return writeTemplate(t, filepath.Join(dir, name), 0640, tmplData)
}

// Recursively crawls template directory for given kind and
// generates files off templates found there.
func (b Builder) LoadWriteTemplates(tmplData interface{}) error {
	sPath := filepath.Join(b.s.TemplatePath, b.kind)
	tPath := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)

	log.Printf("loading/compiling templates for project %s: %s -> %s", b.p.PrettyName(), sPath, tPath)

	if _, err := os.Stat(sPath); os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(tPath); os.IsNotExist(err) {
		if err := os.MkdirAll(tPath, 0755); err != nil {
			return err
		}
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
		log.Printf("creating: %s", dst)

		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}
	}
	for dst, t := range templates {
		if err := writeTemplate(t, dst, 0644, tmplData); err != nil {
			return err
		}
	}
	return nil
}

func loadTemplate(path string) (*template.Template, error) {
	log.Printf("loading template: %s", path)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t := template.New(path) // use this as identifier
	return t.Parse(string(bytes))
}

// FIXME clean up partially written file
func writeTemplate(t *template.Template, dst string, perm os.FileMode, tmplData interface{}) error {
	log.Printf("compiling template to: %s", dst)

	fh, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		return err
	}
	defer fh.Close()
	return t.Execute(fh, tmplData)
}
