// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package project

import (
	"testing"
)

func TestValidBasicRequirements(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate context or webroot")
	}
}

func TestMissingContext(t *testing.T) {
	hoifile := `
webroot = "app/webroot"
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect missing context")
	}
}

func TestMissingWebroot(t *testing.T) {
	hoifile := `
context = "dev"
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect missing webroot")
	}
}

func TestWebrootAbsPath(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "/home/john/app/webroot"
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect webroot path is absolute")
	}
}

func TestValidDomainInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	aliases = ["example.com", "example.net"]
	redirects = ["foo.org", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate domain in prod context")
	}
}

func TestInvalidDomainInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.test {
	aliases = ["example.com", "xmpp.test"]
	redirects = ["foo.test", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect invalid domain in prod context")
	}
}

func TestInvalidAliasInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	aliases = ["example.com", "xmpp.test"]
	redirects = ["foo.org", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect invalid alias in prod context")
	}
}

func TestInvalidRedirectInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	aliases = ["example.com", "xmpp.com"]
	redirects = ["foo.test", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect invalid redirect in prod context")
	}
}

func TestValidDomainInStageContext(t *testing.T) {
	hoifile := `
context = "stage"
webroot = "app/webroot"
domain example.org {
	aliases = ["example.com", "example.net"]
	redirects = ["foo.org", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate domain in stage context")
	}
}

func TestInvalidDomainInStageContext(t *testing.T) {
	hoifile := `
context = "stage"
webroot = "app/webroot"
domain example.test {
	aliases = ["example.com", "xmpp.test"]
	redirects = ["foo.test", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect invalid domain in stage context")
	}
}

func TestInvalidAliasInStageContext(t *testing.T) {
	hoifile := `
context = "stage"
webroot = "app/webroot"
domain example.org {
	aliases = ["example.com", "xmpp.test"]
	redirects = ["foo.org", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect invalid alias in stage context")
	}
}

func TestInvalidRedirectInStageContext(t *testing.T) {
	hoifile := `
context = "stage"
webroot = "app/webroot"
domain example.org {
	aliases = ["example.com", "xmpp.com"]
	redirects = ["foo.test", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect invalid redirect in stage context")
	}
}

func TestValidTestDomainInDevContext(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.test {
	aliases = ["example.com", "xmpp.test"]
	redirects = ["foo.test", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate domain in dev context")
	}
}

func TestValidNonTestDomainInDevContext(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.org {
	aliases = ["example.com", "xmpp.test"]
	redirects = ["foo.test", "bar.org"]
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate non-dev domain in dev context")
	}
}

func TestValidEmptyAuthInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = ""
		password = ""
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate empty auth in prod context")
	}
}

func TestValidFullAuthInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = "john"
		password = "s3cret"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate full auth in prod context")
	}
}

func TestInvalidAuthMissingPasswordInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = "john"
		password = ""
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect missing auth password in prod context")
	}
}

func TestInvalidAuthMissingUsernameInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = ""
		password = "s3cret"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect missing auth username in prod context")
	}
}

func TestValidEmptyAuthInStageContext(t *testing.T) {
	hoifile := `
context = "stage"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = ""
		password = ""
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate empty auth in stage context")
	}
}

func TestValidFullAuthInStageContext(t *testing.T) {
	hoifile := `
context = "stage"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = "john"
		password = "s3cret"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate full auth in stage context")
	}
}

func TestInvalidAuthMissingPasswordInStageContext(t *testing.T) {
	hoifile := `
context = "stage"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = "john"
		password = ""
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect missing auth password in stage context")
	}
}

func TestInvalidAuthMissingUsernameInStageContext(t *testing.T) {
	hoifile := `
context = "stage"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = ""
		password = "s3cret"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect missing auth username in stage context")
	}
}

func TestValidEmptyAuthInDevContext(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = ""
		password = ""
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate empty auth in dev context")
	}
}

func TestValidFullAuthInDevContext(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = "john"
		password = "s3cret"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate full auth in dev context")
	}
}

func TestValidAuthMissingPasswordInDevContext(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = "john"
		password = ""
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate missing auth password in dev context")
	}
}

func TestInvalidAuthMissingUsernameInDevContext(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = ""
		password = "s3cret"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect missing auth username in dev context")
	}
}

func TestValidAuthDuplicateUser(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = "john"
		password = "s3cret"
	}
}
domain another.org {
	auth = {
		user = "john"
		password = "s3cret"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate coherent auth recurrence")
	}
}

func TestInvalidAuthDuplicateUser(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	auth = {
		user = "john"
		password = "s3cret"
	}
}
domain another.org {
	auth = {
		user = "john"
		password = "different_s3cret"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect incoherent auth recurrence")
	}
}

func TestValidEmptySSL(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = ""
		certificateKey = ""
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate empty SSL")
	}
}

func TestValidFullSSL(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = "example.org.crt"
		certificateKey = "example.org.key"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate full SSL")
	}
}

func TestInvalidSSLMissingKey(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = "example.org.crt"
		certificateKey = ""
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect missing SSL key")
	}
}

func TestInvalidSSLMissingCert(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = ""
		certificateKey = "example.org.key"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect missing SSL cert")
	}
}

func TestInvalidSSLSpecialActionMissingCert(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = "example.org.crt"
		certificateKey = "!self-signed"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect SSL special action missing for cert")
	}
}

func TestInvalidSSLSpecialActionMissingKey(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = "!self-signed"
		certificateKey = "example.org.key"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect SSL special action missing for key")
	}
}

func TestInvalidSSLCertPathAbsolute(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = "/home/john/example.org.crt"
		certificateKey = "example.org.key"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect SSL cert path is absolute")
	}
}

func TestInvalidSSLKeyPathAbsolute(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = "example.org.crt"
		certificateKey = "/home/john/example.org.key"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect SSL key path is absolute")
	}
}

func TestValidSSLSelfSignedInDevContext(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = "!self-signed"
		certificateKey = "!self-signed"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate self-signed SSL cert in dev context")
	}
}

func TestInvalidSSLSelfSignedInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {
	SSL = {
		certificate = "!self-signed"
		certificateKey = "!self-signed"
	}
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect self-signed SSL cert in prod context")
	}
}

func TestValidDatabaseInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {}
database example {
	password = "s3cret"
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate database in prod context")
	}
}

func TestInvalidDatabaseWithoutPasswordInProdContext(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {}
database example {
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect database without password in prod context")
	}
}

func TestInvalidDatabaseWithoutPasswordInStageContext(t *testing.T) {
	hoifile := `
context = "stage"
webroot = "app/webroot"
domain example.org {}
database example {
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect database without password in stage context")
	}
}

func TestValidDatabaseInDevContext(t *testing.T) {
	hoifile := `
context = "dev"
webroot = "app/webroot"
domain example.org {}
database example {
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate database in dev context")
	}
}

func TestInvalidDatabaseNameEmpty(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {}
database "" {
	password = "s3cret"	
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect empty database name")
	}
}

func TestInvalidDatabaseDuplicated(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {}
database example {
	password = "s3cret"
}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Database["example2"] = DatabaseDirective{Name: "example"}
	if cfg.Validate() == nil {
		t.Error("failed to detect database name duplicate")
	}
}

func TestValidVolumePath(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {}
volume log {}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() != nil {
		t.Error("failed to validate volume path")
	}
}

func TestInvalidVolumePathAbsolute(t *testing.T) {
	hoifile := `
context = "prod"
webroot = "app/webroot"
domain example.org {}
volume "/etc/log" {}
`
	cfg, err := NewFromString(hoifile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Error("failed to detect absolute volume path")
	}
}
