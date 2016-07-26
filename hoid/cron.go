// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
)

func deactivateCron(pCfg *pConfig.Config, sCfg *sConfig.Config) error {
	runPath, err := sCfg.Systemd.GetRunPath()
	if err != nil {
		return err
	}
	pattern := fmt.Sprintf("project_%s_cron_*", pCfg.Id())

	if os.Getenv("HOI_NOOP") != "yes" {
		if err := exec.Command("systemctl", "disable", "--now", pattern).Run(); err != nil {
			return err
		}
	}

	files, err := filepath.Glob(runPath + "/" + pattern)
	if err != nil {
		return err
	}
	for _, f := range files {
		log.Printf("removing: %s", f)
		if err := os.Remove(f); err != nil {
			return err
		}
	}
	return nil
}

func activateCron(pCfg *pConfig.Config, sCfg *sConfig.Config) error {
	buildPath, err := sCfg.Systemd.GetBuildPathForProject(pCfg)
	if err != nil {
		return err
	}
	runPath, err := sCfg.Systemd.GetRunPath()
	if err != nil {
		return err
	}

	files, err := filepath.Glob(buildPath + "/cron_*")
	if err != nil {
		return err
	}
	for _, f := range files {
		source := fmt.Sprintf("%s/%s", buildPath, filepath.Base(f))
		target := fmt.Sprintf("%s/project_%s_%s", runPath, pCfg.Id(), filepath.Base(f))

		log.Printf("symlinking: %s -> %s", prettyPath(source), prettyPath(target))
		if err := os.Symlink(source, target); err != nil {
			return err
		}

		if os.Getenv("HOI_NOOP") != "yes" {
			if err := exec.Command("systemctl", "start", target).Run(); err != nil {
				return err
			}
		}
	}
	return nil
}

func generateCron(pCfg *pConfig.Config, sCfg *sConfig.Config) error {
	templatePath, err := sCfg.Systemd.GetTemplatePath()
	if err != nil {
		return err
	}
	buildPath, err := sCfg.Systemd.GetBuildPathForProject(pCfg)
	if err != nil {
		return err
	}

	files, err := filepath.Glob(buildPath + "/cron_*")
	if err != nil {
		return err
	}
	for _, f := range files {
		log.Printf("removing: %s", prettyPath(f))
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	tS, err := loadTemplate(templatePath + "/cron.service")
	tT, err := loadTemplate(templatePath + "/cron.timer")
	if err != nil {
		return err
	}
	for k, v := range pCfg.Cron {
		// Each command can also contain template vars.
		// this must happen before creating configuration files, as
		// the command string is used there.
		log.Printf("parsing command template: %s", v.Command)
		cmdTmplData := struct {
			P pConfig.Config
		}{
			P: *pCfg,
		}
		buf := new(bytes.Buffer)
		cmdT := template.New(v.Name)
		cmdT.Parse(v.Command)
		if err := cmdT.Execute(buf, cmdTmplData); err != nil {
			return err
		}
		v.Command = buf.String()

		tmplData := struct {
			P pConfig.Config
			S sConfig.Config
			C pConfig.CronDirective
		}{
			P: *pCfg,
			S: *sCfg,
			C: v,
		}

		log.Printf("creating directory: %s", prettyPath(buildPath))
		if err := os.MkdirAll(buildPath, 0755); err != nil {
			return err
		}

		if err := writeTemplate(tS, buildPath+"/cron_"+k+".service", tmplData); err != nil {
			return err
		}
		if err := writeTemplate(tT, buildPath+"/cron_"+k+".timer", tmplData); err != nil {
			return err
		}
	}
	return nil
}
