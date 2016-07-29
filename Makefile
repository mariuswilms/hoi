# Copyright 2016 Atelier Disko. All rights reserved.
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

PREFIX ?= /usr/local
VERSION ?= 0.1.0
LDFLAGS=-ldflags "-X main.Version=$(VERSION) "
DEBUG ?= no

define TEST_HOIFILE
name = "foo"
context = "stage"
domain atelierdisko.de {
	www = "drop"
	aliases = ["disko.xyz", "disko.io"]
}
seal {
	ignore = [
		"./app/resources/tmp/cache",
		"./media",
		"./media_versions",
		"./app/webroot/media",
		"./log",
		"./tmp"
	]
}
cron high-freq {
	schedule = "*/10 * * * *"
	command = "./bin/li3.php jobs runFrequency high"
}
cron medium-freq {
	schedule = "hourly"
	command = "./bin/li3.php jobs runFrequency medium"
}
worker media-fix {
	command = "./app/libraries/bin/cute-worker --queue=fix --scope={{.P.Name}}_{{.P.Context}} -r app/libraries/base_core/config/bootstrap.php"
}
endef

CONF_FILES = $(patsubst conf/%,$(PREFIX)/etc/hoi/%,$(shell find conf -type f))

.PHONY: install
install: $(PREFIX)/bin/hoictl $(PREFIX)/sbin/hoid $(CONF_FILES)

.PHONY: dist
dist: dist/hoictl dist/hoid

.PHONY: test
export TEST_HOIFILE
test: 
	rm -fr _test/*
	mkdir -p _test/bin 
	mkdir -p _test/sbin 
	mkdir -p _test/var/run
	mkdir -p _test/etc/hoi 
	mkdir -p _test/etc/nginx/sites-enabled 
	mkdir -p _test/etc/systemd/system
	mkdir -p _test/var/www/foo
	mkdir -p _test/var/www/foo/assets 
	mkdir -p _test/var/www/foo/media
	mkdir -p _test/var/www/foo/media_versions
	mkdir -p _test/var/www/foo/app/webroot
	touch _test/var/www/foo/app/webroot/index.php
	echo "$$TEST_HOIFILE" > _test/var/www/foo/Hoifile
	PREFIX=./_test DEBUG=$(DEBUG) make install
	@echo To run manual test do:
	@echo ----------------------
	@echo cd _test
	@echo export HOI_NOOP=yes
	@echo export HOID_SOCKET=var/run/hoid.socket
	@echo sbin/hoid --config=etc/hoi/hoid.conf
	@echo bin/hoictl add var/www/foo
	@echo bin/hoictl enable var/www/foo

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
	sed -i -e "s|__PREFIX__|$(PREFIX)|g" $@
	chmod 600 $@

dist/%: % config/project config/server hoid/rpc
ifeq ($(DEBUG),yes) 
	godebug build -o $@ ./$<
else
	go build $(LDFLAGS) -o $@ ./$<
endif


