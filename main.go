package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
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
	var verbose bool
	var mode string
	var dateFormat string
	var cut int
	var groups stringsFlags
	var codes ints64Flags
	flag.BoolVar(&verbose, "v", false, "verbose mode")
	flag.BoolVar(&inv, "i", false, "invert foreground print color")
	flag.StringVar(&mode, "mode", "url", "how to organize hits url|ua|date")
	flag.StringVar(&dateFormat, "format", "YYYY-MM-DD", "date formatting")
	flag.IntVar(&cut, "cut", 0, "cut some caracters from the beginning of each line")
	flag.Var(&groups, "group", "url group regexp such as [name]=[re]")
	flag.Var(&codes, "code", "only those http codes (comma split)")
	flag.Parse()

	if inv {
		color = goterm.BLACK
	}

	stdin := make(chan []byte)
	go read(stdin, os.Stdin)

	lines := make(chan LogLine)
	workers := 4
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parseLines(cut, verbose, lines, stdin)
		}()
	}
	go func() {
		wg.Wait()
		close(lines)
	}()

	dateFormat = strings.Replace(dateFormat, "YYYY", "2006", -1)
	dateFormat = strings.Replace(dateFormat, "MM", "01", -1)
	dateFormat = strings.Replace(dateFormat, "DD", "02", -1)
	dateFormat = strings.Replace(dateFormat, "hh", "15", -1)
	dateFormat = strings.Replace(dateFormat, "mm", "54", -1)
	dateFormat = strings.Replace(dateFormat, "ii", "05", -1)

	srv := serverProcesser{
		mode:       mode,
		dateFormat: dateFormat,
		groups:     map[string]*regexp.Regexp{},
		only:       map[string]struct{}{},
		stats:      map[string]httpStat{},
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

	ticker := time.Tick(time.Millisecond * 500)
	for {
		select {
		case <-ticker:
			printJobs(srv.Stats())
		case line := <-lines:
			srv.Process(line)
		}
		<-time.After(time.Microsecond * 100)
	}
}

func read(dst chan []byte, src io.Reader) {
	defer close(dst)
	sc := bufio.NewScanner(src)
	for sc.Scan() {
		dst <- sc.Bytes()
	}
}

func parseLines(cut int, verbose bool, dst chan LogLine, src chan []byte) {
	for line := range src {
		parsed, err := Parse(cut, line)
		if err != nil {
			if verbose {
				log.Println(err, string(line))
			}
			continue
		}
		dst <- parsed
	}
}

type serverProcesser struct {
	mode       string
	dateFormat string
	groups     map[string]*regexp.Regexp
	only       map[string]struct{}
	stats      map[string]httpStat
}

type LogLine struct {
	RemoteAddr string
	Username   string
	Date       string
	Method     string
	Query      string
	Path       string
	Proto      string
	Code       string
	Length     string
	Host       string
	UA         string
}

var doubleDash = []byte("--")
var dash = []byte("-")
var bracket = []byte("]")
var ws = []byte(" ")
var quote = []byte("\"")

func indexOf(b []byte, startAt int, search []byte) int {
	if len(b) < startAt || startAt < 0 {
		return -1
	}
	return bytes.Index(b[startAt:], search)
}

func Parse(cut int, line []byte) (ret LogLine, err error) {
	if len(line) < cut {
		return ret, fmt.Errorf("line smaller than cut")
	}
	line = line[cut:]
	if len(line) < 55 {
		return ret, fmt.Errorf("line smaller than 55")
	}
	if bytes.HasPrefix(line, doubleDash) {
		return ret, fmt.Errorf("comment")
	}
	curPos := 0
	//pass ip
	if u := indexOf(line, curPos, dash); u > -1 {
		ret.RemoteAddr = string(line[curPos : curPos+u])
		ret.RemoteAddr = strings.TrimSpace(ret.RemoteAddr)
		curPos += u + 1 + 1 // dash + ws
	} else {
		return ret, fmt.Errorf("ip not found")
	}
	//pass username
	if u := indexOf(line, curPos, dash); u > -1 {
		ret.Username = string(line[curPos : curPos+u])
		ret.Username = strings.TrimSpace(ret.Username)
		curPos += u + 1 + 1 // dash + ws
	} else {
		return ret, fmt.Errorf("username not found")
	}
	//pass date
	if u := indexOf(line, curPos, bracket); u > -1 {
		ret.Date = string(line[curPos+1 : curPos+u])
		curPos += u + 1 + 1 // bracket + ws
	} else {
		return ret, fmt.Errorf("date not found")
	}
	//pass method
	if u := indexOf(line, curPos, ws); u > -1 {
		ret.Method = string(line[curPos : curPos+u])
		ret.Method = strings.Trim(ret.Method, "\"")
		curPos += u + 1 // ws
	} else {
		return ret, fmt.Errorf("method not found")
	}
	if u := indexOf(line, curPos, ws); u > -1 {
		ret.Path = string(line[curPos : curPos+u])
		curPos += u + 1 // ws
	} else {
		return ret, fmt.Errorf("path not found")
	}
	//pass proto
	if u := indexOf(line, curPos, ws); u > -1 {
		ret.Proto = string(line[curPos : curPos+u])
		ret.Proto = strings.TrimSpace(ret.Proto)
		ret.Proto = strings.Trim(ret.Proto, "\"")
		curPos += u + 1 // ws
	} else {
		return ret, fmt.Errorf("proto not found")
	}
	if u := indexOf(line, curPos, ws); u > -1 {
		ret.Code = string(line[curPos : curPos+u])
		ret.Code = strings.Trim(ret.Code, "\"")
		curPos += u + 1 // ws
	} else {
		return ret, fmt.Errorf("code not found")
	}
	if u := indexOf(line, curPos, ws); u > -1 {
		ret.Length = string(line[curPos : curPos+u])
		ret.Length = strings.TrimSpace(ret.Length)
		curPos += u + 1 // ws
	} else if curPos < len(line) {
		ret.Length = string(line[curPos:])
		ret.Length = strings.TrimSpace(ret.Length)
		return ret, nil
	} else {
		return ret, fmt.Errorf("length not found")
	}

	if u := indexOf(line, curPos+1, quote); u > -1 {
		ret.Host = string(line[curPos+1 : curPos+u+1])
		curPos += 1 + u + 1 // quote
	} else {
		return ret, fmt.Errorf("host not found")
	}
	/*
	 */
	if u := indexOf(line, curPos+1+1, quote); u > -1 {
		ret.UA = string(line[curPos+1+1 : curPos+u+1+1])
		ret.UA = strings.TrimSpace(ret.UA)
		ret.UA = strings.Trim(ret.UA, "\"")
	} else if curPos < len(line) {
		ret.UA = string(line[curPos:])
		ret.UA = strings.TrimSpace(ret.UA)
		ret.UA = strings.Trim(ret.UA, "\"")
		return ret, nil
	} else {
		return ret, fmt.Errorf("ua not found")
	}
	return ret, nil
}

func (s serverProcesser) Process(line LogLine) {
	var by string
	if s.mode == "url" {
		by = line.Path
	} else if s.mode == "ua" {
		by = line.UA
	} else if s.mode == "date" {
		x, err := time.Parse("02/Jan/2006:15:04:05 -0700", line.Date)
		if err != nil {
			return
		}
		by = x.Format(s.dateFormat)
	}

	if len(by) < 1 {
		return
	}
	if len(line.Code) < 1 {
		return
	}

	if len(s.groups) == 0 {
		httpStatsMap(s.stats).AddStat(by, line.Code)
		return
	}

	for n, re := range s.groups {
		if re.MatchString(by) {
			httpStatsMap(s.stats).AddStat(n, line.Code)
			return
		}
	}

	httpStatsMap(s.stats).AddStat(by, line.Code)
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
