package gocloc

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"unicode"
)

// ClocLanguage is provide for xml-cloc and json format.
type ClocLanguage struct {
	Name       string `xml:"name,attr" json:"name,omitempty"`
	FilesCount int32  `xml:"files_count,attr" json:"files"`
	Code       int32  `xml:"code,attr" json:"code"`
	Comments   int32  `xml:"comment,attr" json:"comment"`
	Blanks     int32  `xml:"blank,attr" json:"blank"`
}

// Language is a type used to definitions and store statistics for one programming language.
type Language struct {
	Name         string
	lineComments []string
	multiLines   [][]string
	Files        []string
	Code         int32
	Comments     int32
	Blanks       int32
	Total        int32
}

// Languages is an array representation of Language.
type Languages []Language

func (ls Languages) Len() int {
	return len(ls)
}
func (ls Languages) Swap(i, j int) {
	ls[i], ls[j] = ls[j], ls[i]
}
func (ls Languages) Less(i, j int) bool {
	if ls[i].Code == ls[j].Code {
		return ls[i].Name < ls[j].Name
	}
	return ls[i].Code > ls[j].Code
}

var reShebangEnv = regexp.MustCompile(`^#! *(\S+/env) ([a-zA-Z]+)`)
var reShebangLang = regexp.MustCompile(`^#! *[.a-zA-Z/]+/([a-zA-Z]+)`)

// Exts is the definition of the language name, keyed by the extension for each language.
var Exts = map[string]string{
	"go": "Go",
}

var shebang2ext = map[string]string{
	"gosh":    "scm",
	"make":    "make",
	"perl":    "pl",
	"rc":      "plan9sh",
	"python":  "py",
	"ruby":    "rb",
	"escript": "erl",
}

func getShebang(line string) (shebangLang string, ok bool) {
	ret := reShebangEnv.FindAllStringSubmatch(line, -1)
	if ret != nil && len(ret[0]) == 3 {
		shebangLang = ret[0][2]
		if sl, ok := shebang2ext[shebangLang]; ok {
			return sl, ok
		}
		return shebangLang, true
	}

	ret = reShebangLang.FindAllStringSubmatch(line, -1)
	if ret != nil && len(ret[0]) >= 2 {
		shebangLang = ret[0][1]
		if sl, ok := shebang2ext[shebangLang]; ok {
			return sl, ok
		}
		return shebangLang, true
	}

	return "", false
}

func getFileTypeByShebang(path string) (shebangLang string, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return // ignore error
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return
	}
	line = bytes.TrimLeftFunc(line, unicode.IsSpace)

	if len(line) > 2 && line[0] == '#' && line[1] == '!' {
		return getShebang(string(line))
	}
	return
}

func getFileType(path string, _ *ClocOptions) (ext string, ok bool) {
	ext = filepath.Ext(path)

	shebangLang, ok := getFileTypeByShebang(path)
	if ok {
		return shebangLang, true
	}

	if len(ext) >= 2 {
		return ext[1:], true
	}
	return ext, ok
}

// NewLanguage create language data store.
func NewLanguage(name string, lineComments []string, multiLines [][]string) *Language {
	return &Language{
		Name:         name,
		lineComments: lineComments,
		multiLines:   multiLines,
		Files:        []string{},
	}
}

// DefinedLanguages is the type information for mapping language name(key) and NewLanguage.
type DefinedLanguages struct {
	Langs map[string]*Language
}

// GetFormattedString return DefinedLanguages as a human readable string.
func (langs *DefinedLanguages) GetFormattedString() string {
	var buf bytes.Buffer
	printLangs := []string{}
	for _, lang := range langs.Langs {
		printLangs = append(printLangs, lang.Name)
	}
	sort.Strings(printLangs)
	return buf.String()
}

// NewDefinedLanguages create DefinedLanguages.
func NewDefinedLanguages() *DefinedLanguages {
	return &DefinedLanguages{
		Langs: map[string]*Language{
			"Go": NewLanguage("Go", []string{"//"}, [][]string{{"/*", "*/"}}),
		},
	}
}
