package main

import (
	"strings"
	"testing"
)

func TestAssetName(t *testing.T) {
	cases := []struct {
		os, arch, want string
		wantErr        bool
	}{
		{"linux", "amd64", "motd_Linux_x86_64.tar.gz", false},
		{"linux", "arm64", "motd_Linux_arm64.tar.gz", false},
		{"darwin", "amd64", "motd_Darwin_x86_64.tar.gz", false},
		{"darwin", "arm64", "motd_Darwin_arm64.tar.gz", false},
		{"windows", "amd64", "", true},
		{"linux", "386", "", true},
	}
	for _, c := range cases {
		got, err := assetName(c.os, c.arch)
		if c.wantErr {
			if err == nil {
				t.Errorf("assetName(%q, %q): want error, got %q", c.os, c.arch, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("assetName(%q, %q): unexpected error: %v", c.os, c.arch, err)
			continue
		}
		if got != c.want {
			t.Errorf("assetName(%q, %q) = %q, want %q", c.os, c.arch, got, c.want)
		}
	}
}

func TestVersionTag(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "dev"},
		{"dev", "dev"},
		{"0.1.2", "v0.1.2"},
		{"v0.1.2", "v0.1.2"},
		{"1.0.0-rc1", "v1.0.0-rc1"},
	}
	for _, c := range cases {
		if got := versionTag(c.in); got != c.want {
			t.Errorf("versionTag(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestVerifyChecksum(t *testing.T) {
	// SHA256 of "hello\n" is the well-known constant below.
	checksums := strings.Join([]string{
		"5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03  hello.txt",
		"deadbeef  other.bin",
	}, "\n")
	data := []byte("hello\n")

	if err := verifyChecksum([]byte(checksums), "hello.txt", data); err != nil {
		t.Fatalf("verifyChecksum: %v", err)
	}

	if err := verifyChecksum([]byte(checksums), "missing.tar.gz", data); err == nil {
		t.Fatal("verifyChecksum: want error for missing entry, got nil")
	}

	if err := verifyChecksum([]byte(checksums), "other.bin", data); err == nil {
		t.Fatal("verifyChecksum: want error for sha mismatch, got nil")
	}
}
