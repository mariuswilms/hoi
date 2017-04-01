# Copyright 2016 Atelier Disko. All rights reserved.
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

PREFIX ?= /usr/local

VERSION ?= head-$(shell git rev-parse --short HEAD)

HOID_GOFLAGS = -X main.Version=$(VERSION)
HOID_GOFLAGS +=  -X main.SocketPath=$(PREFIX)/var/run/hoid.socket
HOID_GOFLAGS +=  -X main.ConfigPath=$(PREFIX)/etc/hoi/hoid.conf
HOID_GOFLAGS +=  -X main.DataPath=$(PREFIX)/var/lib/hoid.db

HOICTL_GOFLAGS = -X main.Version=$(VERSION)
HOICTL_GOFLAGS +=  -X main.SocketPath=$(PREFIX)/var/run/hoid.socket

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
	# Leave configuration as is, as user might have customized it.

.PHONY: clean
clean:
	if [ -d ./_test ]; then rm -fr ./_test; fi
	if [ -d ./dist ]; then rm -r ./dist; fi
	if [ -f ./hoid/hoid ]; then rm ./hoid/hoid; fi
	if [ -f ./hoictl/hoictl ]; then rm ./hoictl/hoictl; fi

.PHONY: dist
dist: dist/hoictl dist/hoid dist/hoictl-darwin-amd64 dist/hoid-darwin-amd64 dist/hoictl-linux-amd64 dist/hoid-linux-amd64

# Runs all unit tests in sub-packages excluding vendor packages.
.PHONY: unit-tests
unit-tests:
	go test $(shell go list ./... | grep -v vendor)

# IMPORTANT: Run only inside the VM.
ifneq ($(wildcard /vagrant),)
.PHONY: system-tests
export TEST_HOIFILE
system-tests:
	PREFIX= make uninstall
	rm -fr /etc/hoi
	rm -fr /var/www/example
	rm -f /usr/lib/tmpfiles.d/hoi.conf /etc/systemd/system/hoid.service

	VERSION=test PREFIX= make install
	sed -i -e "s|useLegacy = false|useLegacy = true|g" /etc/hoi/hoid.conf
	sed -i -e "s|user = \"hoi\"|user = \"root\"|g" /etc/hoi/hoid.conf
	sed -i -e "s|password = \"s3cret\"|password = \"vagrant\"|g" /etc/hoi/hoid.conf

	mkdir -p /var/www/example
	mkdir -p /var/www/example/config/ssl
	openssl genrsa -out /var/www/example/config/ssl/example.org.key 2048
	openssl req -new -x509 -sha256 -nodes -days 365 \
		-key /var/www/example/config/ssl/example.org.key \
		-out /var/www/example/config/ssl/example.org.crt \
		-subj /C=DE/ST=Hamburg/L=Hamburg/O=None/OU=None/CN=example.org/subjectAltName=DNS.1=www.example.org
	mkdir -p /var/www/example/assets
	mkdir -p /var/www/example/media
	mkdir -p /var/www/example/media_versions
	mkdir -p /var/www/example/app/webroot
	touch /var/www/example/app/webroot/index.php
	echo "$$TEST_HOIFILE" > /var/www/example/Hoifile
endif

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
