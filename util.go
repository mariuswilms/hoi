// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"io"
	"os"
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
	return nil
}
