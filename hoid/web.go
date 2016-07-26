// Copyright 2014 Jim Studt. All rights reserved.
// Copyright 2015 tgic. All rights reserved.
// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"

	pConfig "github.com/atelierdisko/hoi/config/project"
	sConfig "github.com/atelierdisko/hoi/config/server"
)

func deactivateWeb(pCfg *pConfig.Config, sCfg *sConfig.Config) error {
	runPath, err := sCfg.NGINX.GetRunPath()
	if err != nil {
		return err
	}
	pattern := fmt.Sprintf("project_%s_server_*", pCfg.Id())

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
	if os.Getenv("HOI_NOOP") == "yes" {
		return nil
	}
	return exec.Command("systemctl", "reload", "nginx").Run()
}

func activateWeb(pCfg *pConfig.Config, sCfg *sConfig.Config) error {
	buildPath, err := sCfg.NGINX.GetBuildPathForProject(pCfg)
	if err != nil {
		return err
	}
	runPath, err := sCfg.NGINX.GetRunPath()
	if err != nil {
		return err
	}
	files, err := ioutil.ReadDir(buildPath + "/servers")
	if err != nil {
		return err
	}
	for _, f := range files {
		source := fmt.Sprintf("%s/servers/%s", buildPath, f.Name())
		target := fmt.Sprintf("%s/project_%s_server_%s", runPath, pCfg.Id(), f.Name())

		log.Printf("symlinking: %s -> %s", prettyPath(source), prettyPath(target))
		if err := os.Symlink(source, target); err != nil {
			return err
		}
	}
	if os.Getenv("HOI_NOOP") == "yes" {
		return nil
	}
	return exec.Command("systemctl", "reload", "nginx").Run()
}

func generateWeb(pCfg *pConfig.Config, sCfg *sConfig.Config) error {
	templatePath, err := sCfg.NGINX.GetTemplatePath()
	if err != nil {
		return err
	}
	buildPath, err := sCfg.NGINX.GetBuildPathForProject(pCfg)
	if err != nil {
		return err
	}

	log.Printf("removing:  %s", prettyPath(buildPath))
	if err := os.RemoveAll(buildPath); err != nil {
		return err
	}

	// Maps usernames to cleartext passwords.
	creds := make(map[string]string)
	for k, v := range pCfg.Domain {
		if v.Auth.User != "" {
			if v.Auth.Password == "" {
				return fmt.Errorf("auth user %s given but empty password for domain %s", v.Auth.User, v.FQDN)
			}
			if _, hasKey := creds[k]; hasKey {
				if creds[k] == v.Auth.Password {
					return fmt.Errorf("auth user %s given multiple times but with differing passwords for domain %s", v.Auth.User, v.FQDN)
				}
			}
			creds[v.Auth.User] = v.Auth.Password
		}
	}
	if len(creds) != 0 {
		err := generateBasicAuthFile(
			buildPath,
			creds,
		)
		if err != nil {
			return err
		}
	}

	tmplData := struct {
		P             pConfig.Config
		S             sConfig.Config
		WebConfigPath string
	}{
		P: *pCfg,
		S: *sCfg,
		// even though we symlink parts of the build path, config files
		// should not rely on symlinking but reference the original
		// created files
		WebConfigPath: buildPath,
	}
	return generateProjectConfig(
		templatePath,
		buildPath,
		tmplData,
	)
}

// APR1-MD5 is the strongest hash nginx supports for basic auth
func generateBasicAuthFile(bPath string, creds map[string]string) error {
	log.Printf("writing basic auth file with %d entry/entries: %s", len(creds), prettyPath(bPath+"/password"))

	if err := os.MkdirAll(bPath, 0755); err != nil {
		return err
	}
	fh, err := os.OpenFile(bPath+"/password", os.O_CREATE|os.O_RDWR, 0640)
	if err != nil {
		return err
	}
	defer fh.Close()

	salt := generateAPR1Salt()

	for u, p := range creds {
		hash := computeAPR1(p, salt)
		fh.WriteString(fmt.Sprintf("%s:%s\n", u, hash))
	}
	return nil
}

const APR1abc string = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// 8 byte long salt from APR1 alphabet
func generateAPR1Salt() string {
	b := make([]byte, 8)

	for i := range b {
		b[i] = APR1abc[rand.Intn(len(APR1abc))]
	}
	return string(b)
}

// This is the MD5 hashing function out of Apache's htpasswd program. The algorithm
// is insane, but we have to match it. Mercifully I found a PHP variant of it at
//   http://stackoverflow.com/questions/2994637/how-to-edit-htpasswd-using-php
// in an answer. That reads better than the original C, and is easy to instrument.
// We will eventually go back to the original apr_md5.c for inspiration when the
// PHP gets too weird.
// The algorithm makes more sense if you imagine the original authors in a pub,
// drinking beer and rolling dice as the fundamental design process.
//
// This implementation has been copied from:
// https://github.com/tg123/go-htpasswd/blob/master/md5.go
// https://github.com/jimstudt/http-authentication/blob/master/basic/md5.go
func computeAPR1(password string, salt string) string {

	// start with a hash of password and salt
	initBin := md5.Sum([]byte(password + salt + password))

	// begin an initial string with hash and salt
	initText := bytes.NewBufferString(password + "$apr1$" + salt)

	// add crap to the string willy-nilly
	for i := len(password); i > 0; i -= 16 {
		lim := i
		if lim > 16 {
			lim = 16
		}
		initText.Write(initBin[0:lim])
	}

	// add more crap to the string willy-nilly
	for i := len(password); i > 0; i >>= 1 {
		if (i & 1) == 1 {
			initText.WriteByte(byte(0))
		} else {
			initText.WriteByte(password[0])
		}
	}

	// Begin our hashing in earnest using our initial string
	bin := md5.Sum(initText.Bytes())

	n := bytes.NewBuffer([]byte{})

	for i := 0; i < 1000; i++ {
		// prepare to make a new muddle
		n.Reset()

		// alternate password+crap+bin with bin+crap+password
		if (i & 1) == 1 {
			n.WriteString(password)
		} else {
			n.Write(bin[:])
		}

		// usually add the salt, but not always
		if i%3 != 0 {
			n.WriteString(salt)
		}

		// usually add the password but not always
		if i%7 != 0 {
			n.WriteString(password)
		}

		// the back half of that alternation
		if (i & 1) == 1 {
			n.Write(bin[:])
		} else {
			n.WriteString(password)
		}

		// replace bin with the md5 of this muddle
		bin = md5.Sum(n.Bytes())
	}

	// At this point we stop transliterating the PHP code and flip back to
	// reading the Apache source. The PHP uses their base64 library, but that
	// uses the wrong character set so needs to be repaired afterwards and reversed
	// and it is just really weird to read.

	result := bytes.NewBuffer([]byte{})

	// This is our own little similar-to-base64-but-not-quite filler
	fill := func(a byte, b byte, c byte) {
		v := (uint(a) << 16) + (uint(b) << 8) + uint(c) // take our 24 input bits

		for i := 0; i < 4; i++ { // and pump out a character for each 6 bits
			result.WriteByte(APR1abc[v&0x3f])
			v >>= 6
		}
	}

	// The order of these indices is strange, be careful
	fill(bin[0], bin[6], bin[12])
	fill(bin[1], bin[7], bin[13])
	fill(bin[2], bin[8], bin[14])
	fill(bin[3], bin[9], bin[15])
	fill(bin[4], bin[10], bin[5]) // 5?  Yes.
	fill(0, 0, bin[11])

	resultString := string(result.Bytes()[0:22]) // we wrote two extras since we only need 22.

	return fmt.Sprintf("$%s$%s$%s", "apr1", salt, resultString)
}
