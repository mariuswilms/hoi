# Copyright 2016 Atelier Disko. All rights reserved.
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

PREFIX ?= /usr/local
FLAG_PREFIX ?= $(PREFIX)

VERSION ?= head-$(shell git rev-parse --short HEAD)

HOID_GOFLAGS = -X main.Version=$(VERSION)
HOID_GOFLAGS +=  -X main.SocketPath=$(FLAG_PREFIX)/var/run/hoid.socket
HOID_GOFLAGS +=  -X main.ConfigPath=$(FLAG_PREFIX)/etc/hoi/hoid.conf

HOICTL_GOFLAGS = -X main.Version=$(VERSION)
HOICTL_GOFLAGS +=  -X main.SocketPath=$(FLAG_PREFIX)/var/run/hoid.socket

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
cron reporter {
  schedule = "daily"
  command = "/bin/touch /tmp/cron-run"
}
worker media-processor {
  instances = 2
  command = "/bin/ping localhost"
}
# 
# volume tmp {
#   isTemporary = true
# }
# volume media {
# }
endef

CONF_FILES = $(patsubst conf/%,$(PREFIX)/etc/hoi/%,conf/hoid.conf $(shell find conf/templates -type f))

.PHONY: install
install: $(PREFIX)/bin/hoictl $(PREFIX)/sbin/hoid $(CONF_FILES) $(PREFIX)/usr/lib/tmpfiles.d/hoi.conf $(PREFIX)/etc/systemd/system/hoid.service

.PHONY: uninstall
uninstall:
	rm -f $(PREFIX)/bin/hoictl
	rm -f $(PREFIX)/sbin/hoid
	rm -f $(PREFIX)/usr/lib/tmpfiles.d/hoi.conf
	rm -f $(PREFIX)/etc/systemd/system/hoid.service
	# Leave other configuration as is, as user might have customized it.
	# Leave database file.

.PHONY: clean
clean:
	if [ -d ./dist ]; then rm -r ./dist; fi
	if [ -f ./hoid/hoid ]; then rm ./hoid/hoid; fi
	if [ -f ./hoictl/hoictl ]; then rm ./hoictl/hoictl; fi
	rm -rf /tmp/hoi_*

.PHONY: dist
dist: dist/hoictl dist/hoid dist/hoictl-darwin-amd64 dist/hoid-darwin-amd64 dist/hoictl-linux-amd64 dist/hoid-linux-amd64 dist/hoi_$(VERSION)-1-amd64.deb

# Runs all unit tests in sub-packages excluding vendor packages.
.PHONY: unit-tests
unit-tests:
	go test $(shell go list ./... | grep -v vendor)

# IMPORTANT: Run only inside the VM.
ifneq ($(wildcard /vagrant),)
.PHONY: system-tests
export TEST_HOIFILE
system-tests: $(PREFIX)/var/www/example
	PREFIX= make uninstall
	rm -fr /etc/hoi

	VERSION=test PREFIX= make install
	sed -i -e "s|useLegacy = false|useLegacy = true|g" /etc/hoi/hoid.conf
	sed -i -e "s|user = \"hoi\"|user = \"root\"|g" /etc/hoi/hoid.conf
	sed -i -e "s|password = \"s3cret\"|password = \"vagrant\"|g" /etc/hoi/hoid.conf
endif

$(PREFIX)/var/www/example:
	mkdir -p $(PREFIX)/var/www/example
	mkdir -p $(PREFIX)/var/www/example/config/ssl
	openssl genrsa -out $(PREFIX)/var/www/example/config/ssl/example.org.key 2048
	openssl req -new -x509 -sha256 -nodes -days 365 \
		-key $(PREFIX)/var/www/example/config/ssl/example.org.key \
		-out $(PREFIX)/var/www/example/config/ssl/example.org.crt \
		-subj /C=DE/ST=Hamburg/L=Hamburg/O=None/OU=None/CN=example.org/subjectAltName=DNS.1=www.example.org
	mkdir -p $(PREFIX)/var/www/example/assets
	mkdir -p $(PREFIX)/var/www/example/media
	mkdir -p $(PREFIX)/var/www/example/app/webroot
	touch $(PREFIX)/var/www/example/app/webroot/index.php
	echo "$$TEST_HOIFILE" > $(PREFIX)/var/www/example/Hoifile

$(PREFIX)/bin/%: dist/%-$(GOOS)-$(GOARCH)
	install -m 555 $< $@

$(PREFIX)/sbin/%: dist/%-$(GOOS)-$(GOARCH)
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

# Version should have the the package revision suffixed: i.e. 0.5.0-1
dist/hoi_%-amd64.deb: 
	mkdir -p /tmp/hoi_$*-amd64/bin
	mkdir -p /tmp/hoi_$*-amd64/sbin
	mkdir -p /tmp/hoi_$*-amd64/usr/lib/tmpfiles.d
	mkdir -p /tmp/hoi_$*-amd64/etc/hoi
	mkdir -p /tmp/hoi_$*-amd64/etc/systemd/system
	mkdir -p /tmp/hoi_$*-amd64/DEBIAN/
	echo "Package: hoi" >> /tmp/hoi_$*-amd64/DEBIAN/control
	echo "Version: $*" >> /tmp/hoi_$*-amd64/DEBIAN/control
	echo "Architecture: amd64" >> /tmp/hoi_$*-amd64/DEBIAN/control
	echo "Depends: systemd (>= 215)" >> /tmp/hoi_$*-amd64/DEBIAN/control
	echo "Maintainer: Atelier Disko <info@atelierdisko.de>" >> /tmp/hoi_$*-amd64/DEBIAN/control
	echo "Description: Host Orchestration Interface" >> /tmp/hoi_$*-amd64/DEBIAN/control
	echo " Hoi is a program that manages the host by orchestrating other services,"  >> /tmp/hoi_$*-amd64/DEBIAN/control
	echo " so projects can be hosted with the execution of just one command." >> /tmp/hoi_$*-amd64/DEBIAN/control
	VERSION=$* GOOS=linux GOARCH=amd64 FLAG_PREFIX= PREFIX=/tmp/hoi_$*-amd64 make install
	dpkg-deb --build /tmp/hoi_$*-amd64
	cp /tmp/hoi_$*-amd64.deb $@
