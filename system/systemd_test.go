// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import "testing"

func TestParseListInstalledUnits(t *testing.T) {
	out := `project_2c2605b0_worker_document@1.service      not-found inactive dead    project_2c2605b0_worker_document@1.service
project_2c2605b0_worker_media-fix@1.service     loaded    active   running Worker media-fix for project npiece@dev
project_2c2605b0_worker_media-fix@2.service     loaded    active   running Worker media-fix for project npiece@dev
project_2c2605b0_worker_media-fixflux@1.service not-found inactive dead    project_2c2605b0_worker_media-fixflux@1.service
`
	r, _ := parseListUnits(out, "project_2c2605b0_worker", "service")
	if len(r) != 4 {
		t.Fail()
	}
}

func TestParseListInstalledUnitsEmptyOut(t *testing.T) {
	out := ``
	r, _ := parseListUnits(out, "project_2c2605b0_worker", "service")
	if len(r) != 0 {
		t.Fail()
	}
}
