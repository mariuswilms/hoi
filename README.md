# Hoi

[![Build Status](https://travis-ci.org/atelierdisko/hoi.svg?branch=master)](https://travis-ci.org/atelierdisko/hoi)
[![GoDoc](https://godoc.org/github.com/atelierdisko/hoi?status.svg)](https://godoc.org/github.com/atelierdisko/hoi)

## Synopsis

Hoi is a host management program that orchestrates other services, so web
projects can be hosted with the execution of just one command. It automates
setting up several aspects of a project i.e. SSL certificates, databases, cron
jobs and HTTP auth for staging areas.

Aspects of a project are defined by the _Hoifile_, a short and sweet per project
configuration file.

**A note on stability:** Although hoi is in the process of being stablized, 
the configuration format and API might change in backwards incompatible ways until reaching 1.0.

## Reasoning

Hoi has been created to ease hosting of the growing number of Atelier Disko
client web projects. By minimizing project setup costs, we want to enable
ourselves to conduct experiments quickly.

> It's as complicated as you want it to be. - Alexander Haensch, [Source](https://twitter.com/alexander_h/status/751470506503798784)

> First off: Kubernetes *is* a complex system. It does a lot and brings new
abstractions. Those abstractions aren't always justified for all problems. I'm
sure that there are plenty of people using Kubernetes that could get by with
something simpler. - Joe Beda, [Source](https://twitter.com/jbeda/status/993978919907770368)

In architecting hoi we made sure to stay very pragmatic, thus hoi uses a
pre-defined set of established/no-nonsense technologies.

Atelier Disko isn't primarly an infrastructure company, so we don't like to
afford maintaining too ambitious solutions. Resources freed from deliberately
choosing a more classic shared hosting architecture are re-invested into providing
a stable, secure and performant hosting environment with good resource
utilization.

Our projects are primarly Go and PHP-based web applications. 

## What's inside?

Hoi consists of a server (_hoid_) and a client (_hoictl_) to control and
query the server. It features several modules (_runners_) which take
care of the different needs of a project.

- [web](https://godoc.org/github.com/atelierdisko/hoi/runner#WebRunner):
  Serves the project under given domains, taking care of SSL
  certificates and basic auth where needed.

- [app service](https://godoc.org/github.com/atelierdisko/hoi/runner#HTTPBackendRunner):
  Allows to use any service (built with i.e. Go, Node.js or Python) that can
  handle HTTP as a web backend.
  
- [php](https://godoc.org/github.com/atelierdisko/hoi/runner#PHPRunner):
  Safely enables per project PHP(1) settings.
  
- [cron](https://godoc.org/github.com/atelierdisko/hoi/runner#CronRunner):
  Starts cron jobs while reducing resource congestion.

- [worker](https://godoc.org/github.com/atelierdisko/hoi/runner#WorkerRunner):
  Manages long running worker processes with resource controls.
  
- [db](https://godoc.org/github.com/atelierdisko/hoi/runner#DBRunner):
  Creates databases and users with minimum set of privileges.
  
- [volume](https://godoc.org/github.com/atelierdisko/hoi/runner#VolumeRunner):
  Mounts persistent and/or temporary volumes into the project.
  
### Reducing Risk of Human Error

Hoi tries to prevent common mistakes that humans make when managing
many projects on a daily basis. The following is an incomplete list
of measures hoi takes.

- no development TLDs in non development environments (i.e. `.test`)
- no circular domain redirects or invalid re-usage of domain aliases
- projects can't use database root user
- no empty passwords in production environments

### Is it for you?

If your hosting needs are similar and are ready to sacrifice the benefits of
i.e. containers for ease of use or you are running services that are not well
suited for per-project containers (i.e. PHP FPM or MySQL), hoi might also be
something for you.

## Installation

The latest installation packages for Debian and Ubuntu and pre-built
binaries for Linux and Darwin can be downloaded from [the releases
page](https://github.com/atelierdisko/hoi/releases).

For Debian and Ubuntu we provide deb packages, that can be installed
easily. Be sure to pick the latest version and the right architecture.
```
$ curl -L https://github.com/atelierdisko/hoi/releases/download/v0.7.0-beta/hoi_0.7.0-beta-1-amd64.deb 
$ dpkg --install hoi_0.7.0-beta-1-amd64.deb
```

You will need to install from source if your distro isn't supported above, need a yet
unreleased version or want to participate in the development of hoi.
```
$ go get github.com/atelierdisko/hoi
$ cd $GOPATH/src/github.com/atelierdisko/hoi
$ PREFIX= make install
$ cp conf/hoid.service /etc/systemd/system/
$ systemctl enable --now hoid
```

## [Project Configuration](https://godoc.org/github.com/atelierdisko/hoi/project#Config): The Hoifile

The Hoifile defines the needs of a project and provides a minimum
set of configuration. The remaining configuration is discovered
automatically once the project is loaded.

It uses a directive based configuration syntax similar to the NGINX
configuration files.

A minimal Hoifile has 3 lines:
```nginx
name = "example"
context = "prod"
domain "example.org" {}
```

A more advanced Hoifile might look like this:
```nginx
name = "example"
context = "prod"
domain "example.org" {
  SSL = {
    certificate = "config/ssl/example.org.crt"
    certificateKey = "config/ssl/example.org.key"
  }
  aliases = ["example.com", "example.net"]
}
database "example" {
  password = "s3cret"
}
cron "reporter" {
  schedule = "daily"
  command = "bin/compile-report"
}
worker "media-processor" {
  instances = 2
  command = "bin/process-media"
}
volume "media_versions" {}
volume "tmp" {
  isTemporary = true
}
```

### Loading and unloading the Hoifile

Once a project contains a Hoifile, it's loaded with a single command:
```
$ cd /var/www/foo
$ hoictl load
```

The loaded configuration can be further manipulated i.e. by adding an
alias to a domain:
```
$ hoictl domain example.org --alias=example.com
```

### Choosing an App HTTP Backend

Hoi understands 3 different kinds of app HTTP backends: `static`, `php` and
`service`. Hoi will automatically discover the app backend kind and most of
its configuration. If you however wish to fine tune it, you can do so using
the `app` directive.

```nginx
app {
  kind = "php"
  useFrontController = true
}
```

The `service` backend requires you to provide a command, that when
executed, starts a HTTP service on the given port. Specifying host and
port is optional, by default localhost and the next free port is used.
SSL termination happens before it reaches the app.

```nginx
app {
  kind = "service"
  command = "bin/server -l {{.P.App.Host}}:{{.P.App.Port}}"
  # host = "192.168.1.23"
  # port = "8080"
}
```

### SSL Configuration
Using a certificate and key that is contained inside the project.

```nginx
  domain "example.org" {
    SSL = {
      certificate = "config/ssl/example.org.crt"
      certificateKey = "config/ssl/example.org.key"
    }
  }

Here we indicate that certificate and key should be generated
for us automatically.

```nginx
  domain "example.org" {
    SSL = {
      certificate = "!self-signed"
      certificateKey = "!generate"
    }
  }
```

It's also possible to use certificates and keys provided by the system. These
must be whitelisted inside the hoid.conf configuration file. This is especially
useful if you are using wildcard certificates.

Hoifile:
```nginx
  domain "foo.example.org" {
    SSL = {
      certificate = "!system"
      certificateKey = "!system"
    }
  }
```

hoid.conf:
```nginx
  SSL {
    system "*.example.org" {
      certificate = "/etc/ssl/certs/star.example.org.crt"
      certificateKey = "/etc/ssl/private/star.example.org.key"
    }
  }
```

## [Server Configuration](https://godoc.org/github.com/atelierdisko/hoi/server#Config): hoid.conf

### Customizing Service Templates

The templates used by hoid to generate service configuration can be customized,
they reside inside `conf/templates` and use [Go Template syntax](https://golang.org/pkg/text/template/).

## Copyright & License

Hoi is Copyright (c) 2016 Atelier Disko if not otherwise
stated. Use of the source code is governed by a BSD-style
license that can be found in the LICENSE file.

## Versions & Requirements

- The Go language >= 1.5 is required to build the project.
 Must have go vendor support enabled.

- Hoi is continously tested on Linux and Darwin.

- Generally systemd(1) is required. 
  Recent systemd versions are always supported, older ones (i.e. 215) are
  probably supported via the `useLegacy` option.

- The web runner requires nginx(8) and openssl(1).
  NGINX >= 1.9.5 is required, older versions can be used by enabling the
  `useLegacy` option. In legacy mode NGINX will log to STDERR and not use
  syslog/journald and disable HTTP2.

- The PHP runner requires php-fpm(8). 
  If you don't enable this runner you can drop this requirement.

- The DB runner requires mysqld(8) or MariaDB.
  If you don't enable this runner you can drop this requirement. MySQL >= 5.7.6
  or MariaDB >= 10.1.3 are required to use more efficient queries. By enabling
  `useLegacy` older versions may be used.


## Development

Hoi comes with unit tests which can be safely executed as they don't
touch the system itself. The unit tests can be run via:

```
$ make unit-tests
```

Not everything is - yet - covered by unit tests. To conduct system tests
this project comes with a Vagrantfile to boot up a VM. The system tests
should only ever be run inside a VM as they modify the systen they run on.

```
HOST  $ go get github.com/atelierdisko/hoi
HOST  $ cd $GOPATH/src/github.com/atelierdisko/hoi
HOST  $ vagrant up
HOST  $ vagrant ssh
```

```
GUEST $ cd go/src/github.com/atelierdisko/hoi
GUEST $ sudo -i
GUEST $ make system-tests
GUEST $ systemctl start hoid
GUEST $ hoictl --project=/var/www/example load
```
