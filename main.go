package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/buger/goterm"
)

type stringsFlags []string

func (i *stringsFlags) String() string {
	return "my string representation"
}

func (i *stringsFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type ints64Flags []string

func (i *ints64Flags) String() string {
	return "my string representation"
}

func (i *ints64Flags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

//color foreground
var color = goterm.WHITE

func main() {
	var inv bool
	var cut int
	var groups stringsFlags
	var codes ints64Flags
	flag.Var(&groups, "group", "url group regexp such as [name]=[re]")
	flag.Var(&codes, "code", "only those http codes (comma split)")
	flag.BoolVar(&inv, "i", false, "invert foreground print color")
	flag.IntVar(&cut, "cut", 0, "cut some caracters from the beginning of each line")
	flag.Parse()

	if inv {
		color = goterm.BLACK
	}

	stdin := make(chan []byte)
	go read(stdin, os.Stdin)

	srv := serverProcesser{
		cut:    cut,
		groups: map[string]*regexp.Regexp{},
		only:   map[string]struct{}{},
		stats:  map[string]httpStat{},
	}

	for _, g := range groups {
		y := strings.Split(g, "=")
		name := y[0]
		r := y[1]
		x := regexp.MustCompile(r)
		srv.groups[name] = x
	}
	for _, c := range codes {
		srv.only[c] = struct{}{}
	}

	secondTicker := time.Tick(time.Second)
	for {
		select {
		case line := <-stdin:
			srv.Process(line)
		case <-secondTicker:
			printJobs(srv.Stats())
		}
		<-time.After(time.Millisecond)
	}
}

func read(dst chan []byte, src io.Reader) {
	defer close(dst)
	sc := bufio.NewScanner(src)
	for sc.Scan() {
		dst <- sc.Bytes()
	}
	panic("r")
}

type serverProcesser struct {
	cut    int
	groups map[string]*regexp.Regexp
	only   map[string]struct{}
	stats  map[string]httpStat
}

var doubleDash = []byte("--")
var dash = []byte("-")
var bracket = []byte("]")
var ws = []byte(" ")
var backslash = []byte("\"")

func indexOf(b []byte, startAt int, search []byte) int {
	if len(b) < startAt || startAt < 0 {
		return -1
	}
	return bytes.Index(b[startAt:], search)
}

func (s serverProcesser) Parse(line []byte) ([]byte, []byte) {
	if len(line) < s.cut {
		return nil, nil
	}
	line = line[s.cut:]
	if len(line) < 55 {
		return nil, nil
	}
	if bytes.HasPrefix(line, doubleDash) {
		return nil, nil
	}
	curPos := 0
	//pass ip
	if u := indexOf(line, curPos, dash); u > -1 {
		curPos += u + 1 + 1 // dash + ws
	} else {
		return nil, nil
	}
	//pass username
	if u := indexOf(line, curPos, dash); u > -1 {
		curPos += u + 1 + 1 // dash + ws
	} else {
		return nil, nil
	}
	//pass date
	if u := indexOf(line, curPos, bracket); u > -1 {
		curPos += u + 1 + 1 // bracket + ws
	} else {
		return nil, nil
	}
	//pass method
	if u := indexOf(line, curPos, ws); u > -1 {
		curPos += u + 1 // ws
	} else {
		return nil, nil
	}
	// path
	var path []byte
	if u := indexOf(line, curPos, ws); u > -1 {
		if curPos+u < len(line) {
			path = append(path, line[curPos:curPos+u]...)
		}
		curPos += u + 1 // ws
	} else {
		return nil, nil
	}
	//pass proto
	if u := indexOf(line, curPos, ws); u > -1 {
		curPos += u + 1 // ws
	} else {
		return nil, nil
	}
	// code
	var code []byte
	if u := indexOf(line, curPos, ws); u > -1 {
		if curPos+u < len(line) {
			code = append(code, line[curPos:curPos+u]...)
		}
		curPos += u + 1 // ws
	} else {
		return nil, nil
	}
	return path, bytes.Trim(code, "\"")
}

func (s serverProcesser) Process(line []byte) {
	path, code := s.Parse(line)
	if path == nil || code == nil {
		return
	}
	spath := strings.TrimSpace(string(path))
	scode := strings.Trim(string(code), " \"")

	if len(spath) < 1 {
		return
	}
	if len(scode) < 1 {
		return
	}

	if len(s.groups) == 0 {
		httpStatsMap(s.stats).AddStat(spath, scode)
		return
	}

	for n, re := range s.groups {
		if re.Match(path) {
			httpStatsMap(s.stats).AddStat(n, scode)
			return
		}
	}

	httpStatsMap(s.stats).AddStat(spath, scode)
	return
}
func (s serverProcesser) Stats() []httpStat {
	keys := []string{}
	for p := range s.stats {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	res := []httpStat{}
	for _, k := range keys {
		res = append(res, s.stats[k])
	}
	return res
}

type httpStatsMap map[string]httpStat

func (h httpStatsMap) AddStat(path, code string) {
	if _, ok := h[path]; !ok {
		i := httpStat{
			Path:      path,
			TotalHits: 1,
			Codes:     map[string]int64{},
		}
		i.Codes[code] = 1
		h[path] = i
	} else {
		i := h[path]
		i.TotalHits++
		if _, ok := i.Codes[code]; !ok {
			i.Codes[code] = 1
		} else {
			i.Codes[code]++
		}
		h[path] = i
	}
}

type httpStat struct {
	Path      string
	TotalHits int64
	Codes     map[string]int64
}
type httpStats []httpStat

func (h httpStats) Codes() []string {
	m := map[string]struct{}{}
	for _, s := range h {
		for c := range s.Codes {
			m[c] = struct{}{}
		}
	}
	ret := []string{}
	for c := range m {
		ret = append(ret, c)
	}
	sort.Slice(ret, func(i int, j int) bool {
		if ret[i] == "\"200\"" {
			return true
		}
		return ret[i] < ret[j]
	})
	return ret
}

func printJobs(stats []httpStat) {
	goterm.Clear() // Clear current screen
	goterm.MoveCursor(1, 1)
	defer goterm.Flush()
	goterm.Println("Current Time:", time.Now().Format("2006-01-02 15:04:05"))

	if len(stats) == 0 {
		goterm.Println("no data yet")
		return
	}

	codes := httpStats(stats).Codes()
	columns := []string{
		"Path",
		"TotalHits",
	}
	columns = append(columns, codes...)

	for i, s := range columns {
		columns[i] = goterm.Bold(goterm.Color(s, color))
	}

	table := goterm.NewTable(0, goterm.Width()-1, 5, ' ', 0)
	fmt.Fprintf(table, "%s\n", strings.Join(columns, "\t"))

	for _, s := range stats {
		// fullSuccess := job.Count == job.CountSuccess
		printCellString(s.Path, table, false, color)
		printCellInt64(s.TotalHits, table, false, color)
		for _, c := range codes {
			printCellInt64(s.Codes[c], table, false, color)
		}
		fmt.Fprintf(table, "\n")
	}

	goterm.Println(table)
}
func printCellInt64(val int64, table *goterm.Table, isBold bool, color int) {
	printCellString(fmt.Sprint(val), table, isBold, color)
}

func printCellString(text string, table *goterm.Table, isBold bool, color int) {
	fmt.Fprintf(table, "%s\t", format(text, color, isBold))
}

func format(text string, color int, isBold bool) string {
	if isBold {
		return goterm.Bold(goterm.Color(text, color))
	}
	return normal(goterm.Color(text, color))
}

func normal(text string) string {
	return fmt.Sprintf("\033[0m%s\033[0m", text)
}
