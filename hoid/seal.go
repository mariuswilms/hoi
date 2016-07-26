// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

/*
func init() {
	desc := `seals a project by allowing to create and check an integrity spec file`
	Deta.Command("seal", desc, func(cmd *cli.Cmd) {
		path := cmd.String(cli.StringArg{
			Name: "PATH",
			Desc: "files and directories under this path are considered",
		})
		file := cmd.String(cli.StringArg{
			Name: "SPEC",
			Desc: "path to spec file",
		})
		exclude := cmd.String(cli.StringOpt{
			Name: "x exclude",
			Desc: "list of space separated excluded directories and files (wildcards are allowed)",
		})
		var excludes []string

		if *exclude != "" {
			excludes = strings.Split(*exclude, " ")
		}

		cmd.Command("create", "create a new spec file", func(cmd *cli.Cmd) {
			cmd.Action = func() {
				if err := createSpec(*file, *path, excludes); err != nil {
					log.Fatal(err)
				}
			}
		})
		cmd.Command("check", "checks integrity", func(cmd *cli.Cmd) {
			if err := checkSpec(*file, *path, excludes); err != nil {
				log.Fatal(err)
			}
		})
	})
}

func createSpec(file string, path string, excludes []string) error {
	log.Printf("Creating integrity spec at: %s", file)

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	// Excludes changes to timestamps as we will create the manifest
	// locally then upload the files. During this process the timestamps
	// may change but we still want to be able to check the manifest.
	args := []string{"-c", "-K", "cksum,md5digest,nochange", "-p", path}
	for _, x := range excludes {
		args = append(args, "-X")
		args = append(args, x)
	}

	cmd := exec.Command("mtree", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = f
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}
	f.Seek(0, 0)
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	log.Printf("SHA256 fingerprint is: %x", h.Sum(nil))
	return nil
}

func checkSpec(file string, path string, excludes []string) error {
	log.Printf("Checking integrity spec from: %s", file)

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	log.Printf("SHA256 fingerprint is: %x", h.Sum(nil))

	args := []string{"-f", file, "-p", path}
	for _, x := range excludes {
		args = append(args, "-X")
		args = append(args, x)
	}

	cmd := exec.Command("mtree", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
*/
