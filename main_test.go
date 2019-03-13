package main

import (
	"bytes"
	"testing"
)

type expected struct {
	in   string
	path []byte
	code []byte
}

func TestCommonFormat(t *testing.T) {
	expects := []expected{
		expected{
			in:   "ar. 2019-03-12 00:07:19 CET. --",
			path: nil,
			code: nil,
		},
		expected{
			in:   "127.0.0.1 - - [11/Mar/2019:23:59:25 +0100] GET /app/css/fontello.css HTTP/1.0 \"200\" 988",
			path: []byte("/app/css/fontello.css"),
			code: []byte("200"),
		},
	}

	s := serverProcesser{}

	for _, e := range expects {
		path, code := s.Parse([]byte(e.in))
		if e.path == nil && path != nil {
			t.Fatal("expected nil path, ", e.in)
		}
		if e.code == nil && code != nil {
			t.Fatal("expected nil code, ", e.in)
		}
		if bytes.Equal(e.path, path) == false {
			t.Fatal("wanted path=", string(e.path), "got path=", string(path))
		}
		if bytes.Equal(e.code, code) == false {
			t.Fatal("wanted code=", string(e.code), "got code=", string(code))
		}
	}
}

func TestCominedFormat(t *testing.T) {
	expects := []expected{
		expected{
			in:   `::1 - - [13/Mar/2019:15:34:39 +0100] "GET / HTTP/1.1" 200 2496 "" "Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:65.0) Gecko/20100101 Firefox/65.0"`,
			path: []byte("/"),
			code: []byte("200"),
		},
	}

	s := serverProcesser{}

	for _, e := range expects {
		path, code := s.Parse([]byte(e.in))
		if e.path == nil && path != nil {
			t.Fatal("expected nil path, ", e.in)
		}
		if e.code == nil && code != nil {
			t.Fatal("expected nil code, ", e.in)
		}
		if bytes.Equal(e.path, path) == false {
			t.Fatal("wanted path=", string(e.path), "got path=", string(path))
		}
		if bytes.Equal(e.code, code) == false {
			t.Fatal("wanted code=", string(e.code), "got code=", string(code))
		}
	}
}
