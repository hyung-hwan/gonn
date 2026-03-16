package main

import "testing"

func TestParseArgsServerDefaultAddress(t *testing.T) {
	var config cli_config
	var files []string
	var err error

	config = default_config()
	files, err = parse_args([]string{"-S", "man/gonn.1.gonn"}, &config)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}

	if !config.server {
		t.Fatalf("server mode not enabled")
	}
	if config.server_address != "0.0.0.0:1207" {
		t.Fatalf("unexpected default server address: %q", config.server_address)
	}
	if len(files) != 1 || files[0] != "man/gonn.1.gonn" {
		t.Fatalf("unexpected files: %#v", files)
	}
}

func TestParseArgsServerExplicitAddressWithEquals(t *testing.T) {
	var config cli_config
	var files []string
	var err error

	config = default_config()
	files, err = parse_args([]string{"--server=127.0.0.1:8080", "man/gonn.1.gonn"}, &config)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}

	if config.server_address != "127.0.0.1:8080" {
		t.Fatalf("unexpected server address: %q", config.server_address)
	}
	if len(files) != 1 || files[0] != "man/gonn.1.gonn" {
		t.Fatalf("unexpected files: %#v", files)
	}
}

func TestParseArgsServerExplicitAddressSeparateArgument(t *testing.T) {
	var config cli_config
	var files []string
	var err error

	config = default_config()
	files, err = parse_args([]string{"-S", ":8080", "man/gonn.1.gonn"}, &config)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}

	if config.server_address != ":8080" {
		t.Fatalf("unexpected server address: %q", config.server_address)
	}
	if len(files) != 1 || files[0] != "man/gonn.1.gonn" {
		t.Fatalf("unexpected files: %#v", files)
	}
}
