# Copyright 2016 Atelier Disko. All rights reserved.
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

PREFIX ?= /usr/local
FLAGS_PREFIX ?= $(abspath PREFIX)

VERSION ?= head-$(shell git rev-parse --short HEAD)

HOID_GOFLAGS = -X main.Version=$(VERSION)
HOID_GOFLAGS +=  -X main.SocketPath=$(FLAGS_PREFIX)/var/run/hoid.socket
HOID_GOFLAGS +=  -X main.ConfigPath=$(FLAGS_PREFIX)/etc/hoi/hoid.conf
HOID_GOFLAGS +=  -X main.DataPath=$(FLAGS_PREFIX)/var/lib/hoid.db

HOICTL_GOFLAGS = -X main.Version=$(VERSION)
HOICTL_GOFLAGS +=  -X main.SocketPath=$(FLAGS_PREFIX)/var/run/hoid.socket

ANY_DEPS = builder project rpc runner server store system util

define TEST_HOIFILE
name = "example"
context = "prod"

domain example.org {
  SSL = {
    certificate = "config/ssl/example.org.crt"
    certificateKey = "config/ssl/example.org.key"
  }
  aliases = ["example.com", "example.net"]
}
database example {
  password = "s3cret"
}
# cron reporter {
#   schedule = "daily"
#   command = "/bin/touch cron-run"
# }
# worker media-processor {
#   instances = 2
#   command = "/bin/touch worker-run"
# }
# 
# volume tmp {
#   isTemporary = true
# }
# volume log {
#   isTemporary = true
# }
# volume media {
# }
# volume media_versions {
# }

endef

CONF_FILES = $(patsubst conf/%,$(PREFIX)/etc/hoi/%,conf/hoid.conf $(shell find conf/templates -type f))

.PHONY: install
install: $(PREFIX)/bin/hoictl $(PREFIX)/sbin/hoid $(CONF_FILES) $(PREFIX)/usr/lib/tmpfiles.d/hoi.conf $(PREFIX)/etc/systemd/system/hoid.service

.PHONY: uninstall
uninstall:
	rm $(PREFIX)/bin/hoictl
	rm $(PREFIX)/sbin/hoid
	# Leave configuration as is, as user might have customized it.

.PHONY: clean
clean:
	if [ -d ./_test ]; then rm -fr ./_test; fi
	if [ -d ./dist ]; then rm -r ./dist; fi
	if [ -f ./hoid/hoid ]; then rm ./hoid/hoid; fi
	if [ -f ./hoictl/hoictl ]; then rm ./hoictl/hoictl; fi

.PHONY: test-clean
test-clean:
	if [ -d ./_test ]; then rm -fr ./_test; fi

.PHONY: dist
dist: dist/hoictl dist/hoid dist/hoictl-darwin-amd64 dist/hoid-darwin-amd64 dist/hoictl-linux-amd64 dist/hoid-linux-amd64

# The resulting environment is to be executed within a vagrant mounted
# virutal machine. 
.PHONY: test
export TEST_HOIFILE
test: 
	rm -fr _test/*
	mkdir -p _test/bin 
	mkdir -p _test/sbin 
	mkdir -p _test/usr/lib/tmpfiles.d
	mkdir -p _test/var/run
	mkdir -p _test/var/lib
	mkdir -p _test/etc/hoi 
	mkdir -p _test/etc/ssl/{certs,private}
	mkdir -p _test/etc/nginx/sites-enabled 
	mkdir -p _test/etc/systemd/system
	mkdir -p _test/etc/php5/fpm/conf.d
	mkdir -p _test/var/projects
	mkdir -p _test/var/tmp
	mkdir -p _test/var/www/example
	mkdir -p _test/var/www/example/config/ssl
	touch _test/var/www/example/config/ssl/example.org.{crt,key}
	mkdir -p _test/var/www/example/assets 
	mkdir -p _test/var/www/example/media
	mkdir -p _test/var/www/example/media_versions
	mkdir -p _test/var/www/example/app/webroot
	touch _test/var/www/example/app/webroot/index.php
	echo "$$TEST_HOIFILE" > _test/var/www/example/Hoifile
	GOOS=linux GOARCH=amd64 VERSION=test PREFIX=./_test FLAGS_PREFIX=/vagrant/_test make install
	sed -i -e "s|Path = \"|Path = \"/vagrant/_test|g" ./_test/etc/hoi/hoid.conf
	sed -i -e "s|useLegacy = false|useLegacy = true|g" ./_test/etc/hoi/hoid.conf
	sed -i -e "s|user = \"hoi\"|user = \"root\"|g" ./_test/etc/hoi/hoid.conf
	sed -i -e "s|password = \"s3cret\"|password = \"vagrant\"|g" ./_test/etc/hoi/hoid.conf
	@echo 
	@echo Terminal A:
	@echo -----------
	@echo ./_test/sbin/hoid 
	@echo 
	@echo Terminal B:
	@echo -----------
	@echo ./_test/bin/hoictl --project=./_test/var/www/example load

$(PREFIX)/bin/%: dist/%
	install -m 555 $< $@

$(PREFIX)/sbin/%: dist/%
	install -m 555 $< $@

$(PREFIX)/etc/hoi/%: conf/%
	@if [ ! -d $(@D) ]; then mkdir -p $(@D); chmod 775 $(@D); fi
	cp $< $@
	chmod 664 $@

$(PREFIX)/etc/hoi/hoid.conf: conf/hoid.conf
	@if [ ! -d $(@D) ]; then mkdir -p $(@D); chmod 775 $(@D); fi
	cp $< $@
	chmod 600 $@

$(PREFIX)/usr/lib/tmpfiles.d/hoi.conf: conf/hoi-tmpfiles.conf
	cp $< $@

$(PREFIX)/etc/systemd/system/hoid.service: conf/hoid.service
	cp $< $@

dist/%: % $(ANY_DEPS) 
	go build -ldflags "$(HOID_GOFLAGS)" -o $@ ./$<

dist/%-darwin-amd64: % $(ANY_DEPS)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(HOID_GOFLAGS)" -o $@ ./$<

dist/%-linux-amd64: % $(ANY_DEPS)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(HOID_GOFLAGS)" -o $@ ./$<
