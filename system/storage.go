// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atelierdisko/hoi/project"
	"github.com/atelierdisko/hoi/server"
)

func NewStorage(p project.Config, s server.Config) *Storage {
	return &Storage{p: p, s: s}
}

type Storage struct {
	p project.Config
	s server.Config
}

func (sys Storage) Install(volume project.VolumeDirective) error {
	ns := fmt.Sprintf("project_%s", sys.p.ID)

	var src string
	if volume.Kind == project.VolumeKindTemporary {
		src = sys.s.Volume.TemporaryRunPath
	} else if volume.Kind == project.VolumeKindPersistent {
		src = sys.s.Volume.PersistentRunPath
	} else {
		return fmt.Errorf("unknown volume kind %s for: %s", volume.Kind, volume.Path)
	}

	src = filepath.Join(src, ns, volume.Path)
	dst := volume.GetAbsolutePath(sys.p)

	u, err := user.Lookup(sys.s.User)
	if err != nil {
		return err
	}
	uid, _ := strconv.Atoi(u.Uid)

	g, err := user.LookupGroup(sys.s.Group)
	if err != nil {
		return err
	}
	gid, _ := strconv.Atoi(g.Gid)

	// 1. owned by global user and group
	// 2. and have the sticky flag set, so when new files are created owner is the same
	// 3. user and group can read AND write, others cannot do anything
	// 4. perms are persisted even for new files
	if _, err := os.Stat(src); os.IsNotExist(err) {
		os.MkdirAll(src, 1770)
		os.Chown(src, uid, gid)

		if err := exec.Command("setfacl", "-d", "-m", "g::rwx", src).Run(); err != nil {
			return fmt.Errorf("failed to set ACLs on mount source %s: %s", src, err)
		}
	} else {
		log.Printf("reusing volume source: %s", src)
	}
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		os.MkdirAll(dst, 1770)
		os.Chown(src, uid, gid)

		if err := exec.Command("setfacl", "-d", "-m", "g::rwx", dst).Run(); err != nil {
			return fmt.Errorf("failed to set ACLs on mount target %s: %s", dst, err)
		}
	}

	if err := exec.Command("mount", "--bind", src, dst).Run(); err != nil {
		return fmt.Errorf("failed bind mounting %s -> %s: %s", src, dst, err)
	}
	if err := persistBindMount("/etc/fstab", src, dst); err != nil {
		return fmt.Errorf("failed persisting bind mount %s -> %s: %s", src, dst, err)
	}
	return nil
}

// Unmount all volumes under the project path. Uses mount command output
// as mtab is not always accessible (darwin) and we hope that command
// output is more consistent accross OSs.
//
// Does *not* Ensures that when unmounting a bind mounted temporary
// volume, it is emptied. Instead the volume must ensure this by itself.
func (sys Storage) Uninstall(volume project.VolumeDirective) error {
	out, err := exec.Command("mount").Output()
	if err != nil {
		return err
	}
	mounts, err := parseMountOutput(string(out))
	if err != nil {
		return err
	}
	for _, dst := range mounts {
		if !strings.HasPrefix(dst, sys.p.Path) {
			continue
		}
		if err := exec.Command("umount", dst).Run(); err != nil {
			return fmt.Errorf("failed unmounting %s: %s", dst, err)
		}
		if err := unpersistBindMount("/etc/fstab", dst); err != nil {
			return fmt.Errorf("unmounted %s, but failed to unpersist: %s", dst, err)
		}
	}
	return nil
}

// Returns mount sources mapped to mount targets. Special mounts
// are ignored.
//
// Line formats
// 1. linux mount
//   /dev/md2 on / type ext4 (rw,noatime,data=ordered)
// 2. darwin mount
//   /dev/disk1 on / (hfs, NFS exported, local, journaled)
func parseMountOutput(output string) (map[string]string, error) {
	mounts := make(map[string]string, 0)

	for _, line := range strings.Split(output, "\n") {
		// Special mounts (w/o a leading slash) are ignored as
		// we probably do not want to access them anyway.
		if line == "" || string(line[0]) != "/" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return mounts, fmt.Errorf("failed to parse mounts line: '%s'", line)
		}
		mounts[fields[0]] = fields[2]
	}
	return mounts, nil
}

// Persist mount by writing inside /etc/fstab.
//
// We are able to detect _our_ mounts by just looking at the paths,
// thus we do not need any markers. By default we append to the end
// of file.
func persistBindMount(fstab string, src string, dst string) error {
	data, err := ioutil.ReadFile(fstab)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(data), `\n`) {
		// src dst ...
		// /dev/md/3 /home ext4 defaults 0 0
		fields := strings.Fields(line)

		if len(fields) >= 2 {
			if fields[1] == dst {
				if fields[0] == src {
					return nil
				}
				return fmt.Errorf("failed persisting mount, target %s already mounted from other source %s", dst, src)
				// we do not have to check the other way around, as
				// it's OK to mount src to multiple targets.
			}
		}
	}

	fw, err := os.OpenFile(fstab, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fw.Close()

	var lineF string
	if strings.HasSuffix(string(data), "\n") {
		lineF = "%s %s none defaults,bind 0 0\n"
	} else {
		lineF = "\n%s %s none defaults,bind 0 0"
	}

	_, err = fw.WriteString(fmt.Sprintf(lineF, src, dst))
	if err != nil {
		return err
	}

	return fw.Sync()
}

func unpersistBindMount(fstab string, dst string) error {
	data, err := ioutil.ReadFile(fstab)
	if err != nil {
		return err
	}
	var lines []string
	var tainted bool
	for _, line := range strings.Split(string(data), "\n") {
		// src dst ...
		// /dev/md/3 /home ext4 defaults 0 0
		fields := strings.Fields(line)

		if len(fields) >= 2 {
			if fields[1] == dst {
				// do not include the line, remove it
				tainted = true
				continue // allow to remove multiple times, cleaning the file
			}
		}
		lines = append(lines, line)
	}

	if !tainted {
		return nil
	}
	fw, err := os.OpenFile(fstab, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fw.Close()

	hasNL := strings.HasSuffix(string(data), "\n")

	for i, line := range lines {
		if len(lines)-1 == i {
			if line != "" && hasNL {
				line = line + "\n"
			}
		} else {
			line = line + "\n"
		}
		if _, err := fw.WriteString(line); err != nil {
			return err
		}
	}
	return fw.Sync()
}
