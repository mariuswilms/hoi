# Copyright 2016 Atelier Disko. All rights reserved.
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

PREFIX ?= /usr/local
VERSION ?= 0.1.0

HOID_GOFLAGS = -X main.Version=$(VERSION)
HOID_GOFLAGS +=  -X main.SocketPath=$(abspath $(PREFIX)/var/run/hoid.socket)
HOID_GOFLAGS +=  -X main.ConfigPath=$(abspath $(PREFIX)/etc/hoi/hoid.conf)

HOICTL_GOFLAGS = -X main.Version=$(VERSION)
HOICTL_GOFLAGS +=  -X main.SocketPath=$(abspath $(PREFIX)/var/run/hoid.socket)

ANY_DEPS = config/project config/server rpc builder runner system

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
	instances = 2
	command = "./app/libraries/bin/cute-worker --queue=fix --scope={{.P.Name}}_{{.P.Context}} -r app/libraries/base_core/config/bootstrap.php"
}
endef

CONF_FILES = $(patsubst conf/%,$(PREFIX)/etc/hoi/%,$(shell find conf -type f))

.PHONY: install
install: $(PREFIX)/bin/hoictl $(PREFIX)/sbin/hoid $(CONF_FILES)

.PHONY: uninstall
uninstall:
	rm $(PREFIX)/bin/hoictl
	rm $(PREFIX)/sbin/hoid

.PHONY: clean
clean:
	if [ -d ./_test ]; then rm -fr ./_test; fi
	if [ -d ./dist ]; then rm -r ./dist; fi
	if [ -f ./hoid/hoid ]; then rm ./hoid/hoid; fi
	if [ -f ./hoictl/hoictl ]; then rm ./hoictl/hoictl; fi

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
	mkdir -p _test/etc/php/fpm/conf.d
	mkdir -p _test/var/www/foo
	mkdir -p _test/var/www/foo/assets 
	mkdir -p _test/var/www/foo/media
	mkdir -p _test/var/www/foo/media_versions
	mkdir -p _test/var/www/foo/app/webroot
	touch _test/var/www/foo/app/webroot/index.php
	echo "$$TEST_HOIFILE" > _test/var/www/foo/Hoifile
	PREFIX=./_test DEBUG=$(DEBUG) make install
	sed -i -e "s|Path = \"|Path = \"$(abspath ./_test)|g" ./_test/etc/hoi/hoid.conf
	@echo 
	@echo Terminal A:
	@echo -----------
	@echo export HOI_NOOP=yes
	@echo ./_test/sbin/hoid 
	@echo 
	@echo Terminal B:
	@echo -----------
	@echo export HOI_NOOP=yes
	@echo ./_test/bin/hoictl --project=./_test/var/www/foo load

$(PREFIX)/bin/%: dist/%
	install -m 555 $< $@

$(PREFIX)/sbin/%: dist/%
	install -m 555 $< $@

$(PREFIX)/etc/hoi/%: conf/%
	@if [ ! -d $(@D) ]; then mkdir -p $(@D); chmod 775 $(@D); fi
	cp $< $@
	chmod 664 $@

$(PREFIX)/etc/systemd/system/hoid.service: conf/hoid.service
	cp $< $@
	chmod 644 $@

$(PREFIX)/etc/hoi/hoid.conf: conf/hoid.conf
	@if [ ! -d $(@D) ]; then mkdir -p $(@D); chmod 775 $(@D); fi
	cp $< $@
	chmod 600 $@

dist/hoid: hoid $(ANY_DEPS) 
	go build -ldflags "$(HOID_GOFLAGS)" -o $@ ./$<

dist/hoictl: hoictl $(ANY_DEPS)
	go build -ldflags "$(HOICTL_GOFLAGS)" -o $@ ./$<


