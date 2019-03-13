package main

import (
	"testing"
)

type expected struct {
	LogLine
	in string
}

func TestCommonFormat(t *testing.T) {
	expects := []expected{
		expected{
			in:      "ar. 2019-03-12 00:07:19 CET. --",
			LogLine: LogLine{},
		},
		expected{
			in: "127.0.0.1 - - [11/Mar/2019:23:59:25 +0100] GET /app/css/fontello.css HTTP/1.0 \"200\" 988",
			LogLine: LogLine{
				Path:       "/app/css/fontello.css",
				Proto:      "HTTP/1.0",
				RemoteAddr: "127.0.0.1",
				Date:       "11/Mar/2019:23:59:25 +0100",
				Host:       "",
				Length:     "988",
				Method:     "GET",
				Code:       "200",
			},
		},
		expected{
			in: `127.0.0.1 - - [13/Mar/2019:17:48:37 +0100] GET /app/master.min.css HTTP/1.0 "200" 15103`,
			LogLine: LogLine{
				Path:       "/app/master.min.css",
				Proto:      "HTTP/1.0",
				RemoteAddr: "127.0.0.1",
				Date:       "13/Mar/2019:17:48:37 +0100",
				Host:       "",
				Length:     "15103",
				Method:     "GET",
				Code:       "200",
			},
		},
	}

	for _, e := range expects {
		line, err := parse(0, []byte(e.in))
		if err != nil {
			t.Log(err)
		}
		compare(t, line, e.LogLine)
	}
}

func TestCominedFormat(t *testing.T) {
	expects := []expected{
		expected{
			in: `::1 - - [13/Mar/2019:15:34:39 +0100] "GET / HTTP/1.1" 200 2496 "" "Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:65.0) Gecko/20100101 Firefox/65.0"`,
			LogLine: LogLine{
				Path:       "/",
				Code:       "200",
				Proto:      "HTTP/1.1",
				RemoteAddr: "::1",
				Date:       "13/Mar/2019:15:34:39 +0100",
				Host:       "",
				UA:         "Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:65.0) Gecko/20100101 Firefox/65.0",
				Length:     "2496",
				Method:     "GET",
			},
		},
		expected{
			in: `82.221.128.136 - - [13/Mar/2019:18:12:23 +0100] "GET /protests/7 HTTP/1.0" 200 1274 "http://monparcours.online/" "Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:65.0) Gecko/20100101 Firefox/65.0"`,
			LogLine: LogLine{
				Path:       "/protests/7",
				Code:       "200",
				Proto:      "HTTP/1.0",
				RemoteAddr: "82.221.128.136",
				Date:       "13/Mar/2019:18:12:23 +0100",
				Host:       "http://monparcours.online/",
				UA:         "Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:65.0) Gecko/20100101 Firefox/65.0",
				Length:     "1274",
				Method:     "GET",
			},
		},
		expected{
			in: `43 t'" t'" t zer zer ezr ez rez rezr 'r43 '(-('v'"é' "é ('('- "`,
			LogLine: LogLine{
				Username:   `'v'"é' "é ('('`,
				RemoteAddr: `43 t'" t'" t zer zer ezr ez rez rezr 'r43 '(`,
			},
		},
	}

	for _, e := range expects {
		line, err := parse(0, []byte(e.in))
		if err != nil {
			t.Log(err)
		}
		compare(t, line, e.LogLine)
	}
}

func compare(t *testing.T, line, e LogLine) {

	if line.UA != e.UA {
		t.Fatal("wanted UA=", (e.UA), "got UA=", (line.UA))
	}
	if line.Host != e.Host {
		t.Fatal("wanted Host=", (e.Host), "got Host=", (line.Host))
	}
	if line.Length != e.Length {
		t.Fatal("wanted Length=", (e.Length), "got Length=", (line.Length))
	}
	if line.Code != e.Code {
		t.Fatal("wanted Code=", (e.Code), "got Code=", (line.Code))
	}
	if line.Proto != e.Proto {
		t.Fatal("wanted Proto=", (e.Proto), "got Proto=", (line.Proto))
	}
	if line.Path != e.Path {
		t.Fatal("wanted Path=", (e.Path), "got Path=", (line.Path))
	}
	if line.RemoteAddr != e.RemoteAddr {
		t.Fatal("wanted RemoteAddr=", (e.RemoteAddr), "got RemoteAddr=", (line.RemoteAddr))
	}
	if line.Username != e.Username {
		t.Fatal("wanted Username=", (e.Username), "got Username=", (line.Username))
	}
	if line.Date != e.Date {
		t.Fatal("wanted Date=", (e.Date), "got Date=", (line.Date))
	}
	if line.Method != e.Method {
		t.Fatal("wanted Method=", (e.Method), "got Method=", (line.Method))
	}
	if line.Query != e.Query {
		t.Fatal("wanted Query=", (e.Query), "got Query=", (line.Query))
	}
}
