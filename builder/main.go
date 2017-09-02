// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builder

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

const (
	KindWeb        = "web"
	KindAppService = "app_service"
	KindPHP        = "php"
	KindCron       = "cron"
	KindWorker     = "worker"
	KindVolume     = "volume"
)

func NewBuilder(kind string, p *project.Config, s *server.Config) *Builder {
	return &Builder{kind: kind, p: p, s: s}
}
func NewScopedBuilder(kind string, scope string, p *project.Config, s *server.Config) *Builder {
	return &Builder{kind: kind, scope: scope, p: p, s: s}
}

// Builds configuration supporting runners.
type Builder struct {
	kind  string
	scope string
	s     *server.Config
	p     *project.Config
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

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed cleaning build directory %s: %s", dir, err)
	}
	return nil
}

func (b Builder) WriteFile(name string, reader io.Reader) error {
	dir := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed writing build file %s, failed, to create dir %s: %s", name, dir, err)
		}
	}
	writer, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed opening build file %s for writing: %s", name, err)
	}
	defer writer.Close()

	if _, err = io.Copy(writer, reader); err != nil {
		return fmt.Errorf("failed copying contents to write build file %s: %s", name, err)
	}
	return nil
}

func (b Builder) LoadTemplate(name string) (*template.Template, error) {
	return loadTemplate(filepath.Join(b.s.TemplatePath, b.kind, name))
}

func (b Builder) WriteTemplate(name string, t *template.Template, tmplData interface{}) error {
	dir := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed writing template %s, failed to create dir %s: %s", name, dir, err)
		}
	}
	return writeTemplate(t, filepath.Join(dir, name), 0644, tmplData)
}

func (b Builder) WriteSensitiveTemplate(name string, t *template.Template, tmplData interface{}) error {
	dir := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed writing sensitive template %s, failed to create dir %s: %s", name, dir, err)
		}
	}
	return writeTemplate(t, filepath.Join(dir, name), 0640, tmplData)
}

// Recursively crawls template directory for given kind and
// generates files off templates found there.
func (b Builder) LoadWriteTemplates(tmplData interface{}) error {
	sPath := filepath.Join(b.s.TemplatePath, b.kind)
	tPath := filepath.Join(b.s.BuildPath, b.kind, b.p.ID)

	if _, err := os.Stat(sPath); os.IsNotExist(err) {
		return fmt.Errorf("failed to prepare loading templates from non-existent path %s: %s", sPath, err)
	}
	if _, err := os.Stat(tPath); os.IsNotExist(err) {
		if err := os.MkdirAll(tPath, 0755); err != nil {
			return fmt.Errorf("failed to prepare writing templates, cannot create directory %s: %s", tPath, err)
		}
	}

	exchanger := strings.NewReplacer(sPath, tPath) // exchange bases
	dirs := make(map[string]string)
	files := make(map[string]string)
	templates := make(map[string]*template.Template) // Keyed by target path.

	err := filepath.Walk(sPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		xPath := exchanger.Replace(path)

		if f.IsDir() {
			dirs[xPath] = path
		} else {
			files[xPath] = path
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed loading templates from %s, cannot map: %s", sPath, err)
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
		if err := os.MkdirAll(dst, 0755); err != nil {
			fmt.Errorf("failed to create nested template target directory %s: %s", dst, err)
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
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load template %s: %s", path, err)
	}
	t := template.New(path) // use this as identifier

	parsed, err := t.Parse(string(bytes))
	if err != nil {
		return nil, fmt.Errorf("tried to load template %s, but it failed to parse: %s", path, err)
	}
	return parsed, nil
}

func writeTemplate(t *template.Template, dst string, perm os.FileMode, tmplData interface{}) error {
	fh, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		return fmt.Errorf("failed to compile template, cannot open target for writing %s: %s", dst, err)
	}
	defer fh.Close()

	if err := t.Execute(fh, tmplData); err != nil {
		execErr := fmt.Errorf("failed to compile template into target %s: %s", dst, err)

		// when execute was aborted while streaming the template, we
		// have a partially written file at hand
		if _, err := os.Stat(dst); err == nil {
			if err := os.Remove(dst); err != nil {
				return fmt.Errorf("failed to clean up after: %s", execErr)
			}
		}
		return execErr
	}
	return nil
}
