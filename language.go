package gocloc

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	enry "github.com/go-enry/go-enry/v2"
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

func getFileType(path string, opts *ClocOptions) (ext string, ok bool) {
	ext = filepath.Ext(path)
	base := filepath.Base(path)

	switch ext {
	case ".m", ".v", ".fs", ".r", ".ts":
		content, err := os.ReadFile(path)
		if err != nil {
			return "", false
		}
		lang := enry.GetLanguage(path, content)
		if opts.Debug {
			fmt.Printf("path=%v, lang=%v\n", path, lang)
		}
		return lang, true
	case ".mo":
		content, err := os.ReadFile(path)
		if err != nil {
			return "", false
		}
		lang := enry.GetLanguage(path, content)
		if opts.Debug {
			fmt.Printf("path=%v, lang=%v\n", path, lang)
		}
		if lang != "" {
			return "Motoko", true
		}
		return lang, true
	}

	switch base {
	case "meson.build", "meson_options.txt":
		return "meson", true
	case "CMakeLists.txt":
		return "cmake", true
	case "configure.ac":
		return "m4", true
	case "Makefile.am":
		return "makefile", true
	case "build.xml":
		return "Ant", true
	case "pom.xml":
		return "maven", true
	}

	switch strings.ToLower(base) {
	case "makefile":
		return "makefile", true
	case "nukefile":
		return "nu", true
	case "rebar": // skip
		return "", false
	}

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

func lang2exts(lang string) (exts string) {
	es := []string{}
	for ext, l := range Exts {
		if lang == l {
			switch lang {
			case "Objective-C", "MATLAB", "Mercury":
				ext = "m"
			case "F#":
				ext = "fs"
			case "GLSL":
				if ext == "GLSL" {
					ext = "fs"
				}
			case "TypeScript":
				ext = "ts"
			case "Motoko":
				ext = "mo"
			}
			es = append(es, ext)
		}
	}
	return strings.Join(es, ", ")
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
	for _, lang := range printLangs {
		buf.WriteString(fmt.Sprintf("%-30v (%s)\n", lang, lang2exts(lang)))
	}
	return buf.String()
}

// NewDefinedLanguages create DefinedLanguages.
func NewDefinedLanguages() *DefinedLanguages {
	return &DefinedLanguages{
		Langs: map[string]*Language{
			"ActionScript":        NewLanguage("ActionScript", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Ada":                 NewLanguage("Ada", []string{"--"}, [][]string{{"", ""}}),
			"Ant":                 NewLanguage("Ant", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"AsciiDoc":            NewLanguage("AsciiDoc", []string{}, [][]string{{"", ""}}),
			"Assembly":            NewLanguage("Assembly", []string{"//", ";", "#", "@", "|", "!"}, [][]string{{"/*", "*/"}}),
			"ATS":                 NewLanguage("ATS", []string{"//"}, [][]string{{"/*", "*/"}, {"(*", "*)"}}),
			"AutoHotkey":          NewLanguage("AutoHotkey", []string{";"}, [][]string{{"", ""}}),
			"Awk":                 NewLanguage("Awk", []string{"#"}, [][]string{{"", ""}}),
			"Arduino Sketch":      NewLanguage("Arduino Sketch", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Batch":               NewLanguage("Batch", []string{"REM", "rem"}, [][]string{{"", ""}}),
			"BASH":                NewLanguage("BASH", []string{"#"}, [][]string{{"", ""}}),
			"BitBake":             NewLanguage("BitBake", []string{"#"}, [][]string{{"", ""}}),
			"C":                   NewLanguage("C", []string{"//"}, [][]string{{"/*", "*/"}}),
			"C Header":            NewLanguage("C Header", []string{"//"}, [][]string{{"/*", "*/"}}),
			"C Shell":             NewLanguage("C Shell", []string{"#"}, [][]string{{"", ""}}),
			"Cairo":               NewLanguage("Cairo", []string{"//"}, [][]string{{"", ""}}),
			"Carbon":              NewLanguage("Carbon", []string{"//"}, [][]string{{"", ""}}),
			"Cap'n Proto":         NewLanguage("Cap'n Proto", []string{"#"}, [][]string{{"", ""}}),
			"Carp":                NewLanguage("Carp", []string{";"}, [][]string{{"", ""}}),
			"C#":                  NewLanguage("C#", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Chapel":              NewLanguage("Chapel", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Clojure":             NewLanguage("Clojure", []string{"#", "#_"}, [][]string{{"", ""}}),
			"COBOL":               NewLanguage("COBOL", []string{"*", "/"}, [][]string{{"", ""}}),
			"CoffeeScript":        NewLanguage("CoffeeScript", []string{"#"}, [][]string{{"###", "###"}}),
			"Coq":                 NewLanguage("Coq", []string{"(*"}, [][]string{{"(*", "*)"}}),
			"ColdFusion":          NewLanguage("ColdFusion", []string{}, [][]string{{"<!---", "--->"}}),
			"ColdFusion CFScript": NewLanguage("ColdFusion CFScript", []string{"//"}, [][]string{{"/*", "*/"}}),
			"CMake":               NewLanguage("CMake", []string{"#"}, [][]string{{"", ""}}),
			"C++":                 NewLanguage("C++", []string{"//"}, [][]string{{"/*", "*/"}}),
			"C++ Header":          NewLanguage("C++ Header", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Crystal":             NewLanguage("Crystal", []string{"#"}, [][]string{{"", ""}}),
			"CSS":                 NewLanguage("CSS", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Cython":              NewLanguage("Cython", []string{"#"}, [][]string{{"\"\"\"", "\"\"\""}}),
			"CUDA":                NewLanguage("CUDA", []string{"//"}, [][]string{{"/*", "*/"}}),
			"D":                   NewLanguage("D", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Dart":                NewLanguage("Dart", []string{"//", "///"}, [][]string{{"/*", "*/"}}),
			"Dhall":               NewLanguage("Dhall", []string{"--"}, [][]string{{"{-", "-}"}}),
			"DTrace":              NewLanguage("DTrace", []string{}, [][]string{{"/*", "*/"}}),
			"Device Tree":         NewLanguage("Device Tree", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Eiffel":              NewLanguage("Eiffel", []string{"--"}, [][]string{{"", ""}}),
			"Elm":                 NewLanguage("Elm", []string{"--"}, [][]string{{"{-", "-}"}}),
			"Elixir":              NewLanguage("Elixir", []string{"#"}, [][]string{{"", ""}}),
			"Erlang":              NewLanguage("Erlang", []string{"%"}, [][]string{{"", ""}}),
			"Expect":              NewLanguage("Expect", []string{"#"}, [][]string{{"", ""}}),
			"Fish":                NewLanguage("Fish", []string{"#"}, [][]string{{"", ""}}),
			"Frege":               NewLanguage("Frege", []string{"--"}, [][]string{{"{-", "-}"}}),
			"F*":                  NewLanguage("F*", []string{"(*", "//"}, [][]string{{"(*", "*)"}}),
			"F#":                  NewLanguage("F#", []string{"(*"}, [][]string{{"(*", "*)"}}),
			"Lean":                NewLanguage("Lean", []string{"--"}, [][]string{{"/-", "-/"}}),
			"Logtalk":             NewLanguage("Logtalk", []string{"%"}, [][]string{{"", ""}}),
			"Lua":                 NewLanguage("Lua", []string{"--"}, [][]string{{"--[[", "]]"}}),
			"Lilypond":            NewLanguage("Lilypond", []string{"%"}, [][]string{{"", ""}}),
			"LISP":                NewLanguage("LISP", []string{";;"}, [][]string{{"#|", "|#"}}),
			"LiveScript":          NewLanguage("LiveScript", []string{"#"}, [][]string{{"/*", "*/"}}),
			"Factor":              NewLanguage("Factor", []string{"! "}, [][]string{{"", ""}}),
			"FORTRAN Legacy":      NewLanguage("FORTRAN Legacy", []string{"c", "C", "!", "*"}, [][]string{{"", ""}}),
			"FORTRAN Modern":      NewLanguage("FORTRAN Modern", []string{"!"}, [][]string{{"", ""}}),
			"Gherkin":             NewLanguage("Gherkin", []string{"#"}, [][]string{{"", ""}}),
			"GLSL":                NewLanguage("GLSL", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Go":                  NewLanguage("Go", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Groovy":              NewLanguage("Groovy", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Handlebars":          NewLanguage("Handlebars", []string{}, [][]string{{"<!--", "-->"}, {"{{!", "}}"}}),
			"Haskell":             NewLanguage("Haskell", []string{"--"}, [][]string{{"{-", "-}"}}),
			"Haxe":                NewLanguage("Haxe", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Hare":                NewLanguage("Hare", []string{"//"}, [][]string{{"", ""}}),
			"HLSL":                NewLanguage("HLSL", []string{"//"}, [][]string{{"/*", "*/"}}),
			"HTML":                NewLanguage("HTML", []string{"//", "<!--"}, [][]string{{"<!--", "-->"}}),
			"Idris":               NewLanguage("Idris", []string{"--"}, [][]string{{"{-", "-}"}}),
			"Imba":                NewLanguage("Imba", []string{"#"}, [][]string{{"###", "###"}}),
			"Io":                  NewLanguage("Io", []string{"//", "#"}, [][]string{{"/*", "*/"}}),
			"SKILL":               NewLanguage("SKILL", []string{";"}, [][]string{{"/*", "*/"}}),
			"JAI":                 NewLanguage("JAI", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Janet":               NewLanguage("Janet", []string{"#"}, [][]string{{"", ""}}),
			"Java":                NewLanguage("Java", []string{"//"}, [][]string{{"/*", "*/"}}),
			"JSP":                 NewLanguage("JSP", []string{"//"}, [][]string{{"/*", "*/"}}),
			"JavaScript":          NewLanguage("JavaScript", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Julia":               NewLanguage("Julia", []string{"#"}, [][]string{{"#:=", ":=#"}}),
			"Jupyter Notebook":    NewLanguage("Jupyter Notebook", []string{"#"}, [][]string{{"", ""}}),
			"JSON":                NewLanguage("JSON", []string{}, [][]string{{"", ""}}),
			"JSX":                 NewLanguage("JSX", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Koka":                NewLanguage("Koka", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Kotlin":              NewLanguage("Kotlin", []string{"//"}, [][]string{{"/*", "*/"}}),
			"LD Script":           NewLanguage("LD Script", []string{"//"}, [][]string{{"/*", "*/"}}),
			"LESS":                NewLanguage("LESS", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Objective-C":         NewLanguage("Objective-C", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Markdown":            NewLanguage("Markdown", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"Motoko":              NewLanguage("Motoko", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Nix":                 NewLanguage("Nix", []string{"#"}, [][]string{{"/*", "*/"}}),
			"NSIS":                NewLanguage("NSIS", []string{"#", ";"}, [][]string{{"/*", "*/"}}),
			"Nu":                  NewLanguage("Nu", []string{";", "#"}, [][]string{{"", ""}}),
			"OCaml":               NewLanguage("OCaml", []string{}, [][]string{{"(*", "*)"}}),
			"Objective-C++":       NewLanguage("Objective-C++", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Makefile":            NewLanguage("Makefile", []string{"#"}, [][]string{{"", ""}}),
			"MATLAB":              NewLanguage("MATLAB", []string{"%"}, [][]string{{"%{", "}%"}}),
			"Mercury":             NewLanguage("Mercury", []string{"%"}, [][]string{{"/*", "*/"}}),
			"Maven":               NewLanguage("Maven", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"Meson":               NewLanguage("Meson", []string{"#"}, [][]string{{"", ""}}),
			"Mojo":                NewLanguage("Mojo", []string{"#"}, [][]string{{"", ""}}),
			"Move":                NewLanguage("Move", []string{"//"}, [][]string{{"", ""}}),
			"Mustache":            NewLanguage("Mustache", []string{}, [][]string{{"{{!", "}}"}}),
			"M4":                  NewLanguage("M4", []string{"#"}, [][]string{{"", ""}}),
			"Nim":                 NewLanguage("Nim", []string{"#"}, [][]string{{"#[", "]#"}}),
			"Nunjucks":            NewLanguage("Nunjucks", []string{}, [][]string{{"{#", "#}"}, {"<!--", "-->"}}),
			"lex":                 NewLanguage("lex", []string{}, [][]string{{"/*", "*/"}}),
			"Odin":                NewLanguage("Odin", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Ohm":                 NewLanguage("Ohm", []string{"//"}, [][]string{{"/*", "*/"}}),
			"PHP":                 NewLanguage("PHP", []string{"#", "//"}, [][]string{{"/*", "*/"}}),
			"Pascal":              NewLanguage("Pascal", []string{"//"}, [][]string{{"{", ")"}}),
			"Perl":                NewLanguage("Perl", []string{"#"}, [][]string{{":=", ":=cut"}}),
			"Plain Text":          NewLanguage("Plain Text", []string{}, [][]string{{"", ""}}),
			"Plan9 Shell":         NewLanguage("Plan9 Shell", []string{"#"}, [][]string{{"", ""}}),
			"Pony":                NewLanguage("Pony", []string{"//"}, [][]string{{"/*", "*/"}}),
			"PowerShell":          NewLanguage("PowerShell", []string{"#"}, [][]string{{"<#", "#>"}}),
			"Polly":               NewLanguage("Polly", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"Protocol Buffers":    NewLanguage("Protocol Buffers", []string{"//"}, [][]string{{"", ""}}),
			"Python":              NewLanguage("Python", []string{"#"}, [][]string{{"\"\"\"", "\"\"\""}}),
			"Q":                   NewLanguage("Q", []string{"/ "}, [][]string{{"\\", "/"}, {"/", "\\"}}),
			"QML":                 NewLanguage("QML", []string{"//"}, [][]string{{"/*", "*/"}}),
			"R":                   NewLanguage("R", []string{"#"}, [][]string{{"", ""}}),
			"Rebol":               NewLanguage("Rebol", []string{";"}, [][]string{{"", ""}}),
			"Red":                 NewLanguage("Red", []string{";"}, [][]string{{"", ""}}),
			"Rego":                NewLanguage("Rego", []string{"#"}, [][]string{{"", ""}}),
			"RMarkdown":           NewLanguage("RMarkdown", []string{}, [][]string{{"", ""}}),
			"RAML":                NewLanguage("RAML", []string{"#"}, [][]string{{"", ""}}),
			"Racket":              NewLanguage("Racket", []string{";"}, [][]string{{"#|", "|#"}}),
			"ReStructuredText":    NewLanguage("ReStructuredText", []string{}, [][]string{{"", ""}}),
			"Ring":                NewLanguage("Ring", []string{"#", "//"}, [][]string{{"/*", "*/"}}),
			"Ruby":                NewLanguage("Ruby", []string{"#"}, [][]string{{":=begin", ":=end"}}),
			"Ruby HTML":           NewLanguage("Ruby HTML", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"Rust":                NewLanguage("Rust", []string{"//", "///", "//!"}, [][]string{{"/*", "*/"}}),
			"Scala":               NewLanguage("Scala", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Sass":                NewLanguage("Sass", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Scheme":              NewLanguage("Scheme", []string{";"}, [][]string{{"#|", "|#"}}),
			"sed":                 NewLanguage("sed", []string{"#"}, [][]string{{"", ""}}),
			"Stan":                NewLanguage("Stan", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Solidity":            NewLanguage("Solidity", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Bourne Shell":        NewLanguage("Bourne Shell", []string{"#"}, [][]string{{"", ""}}),
			"Standard ML":         NewLanguage("Standard ML", []string{}, [][]string{{"(*", "*)"}}),
			"SQL":                 NewLanguage("SQL", []string{"--"}, [][]string{{"/*", "*/"}}),
			"Svelte":              NewLanguage("Svelte", []string{"//"}, [][]string{{"/*", "*/"}, {"<!--", "-->"}}),
			"Swift":               NewLanguage("Swift", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Terra":               NewLanguage("Terra", []string{"--"}, [][]string{{"--[[", "]]"}}),
			"TeX":                 NewLanguage("TeX", []string{"%"}, [][]string{{"", ""}}),
			"Isabelle":            NewLanguage("Isabelle", []string{}, [][]string{{"(*", "*)"}}),
			"TLA":                 NewLanguage("TLA", []string{"\\*"}, [][]string{{"(*", "*)"}}),
			"Tcl/Tk":              NewLanguage("Tcl/Tk", []string{"#"}, [][]string{{"", ""}}),
			"TOML":                NewLanguage("TOML", []string{"#"}, [][]string{{"", ""}}),
			"TypeScript":          NewLanguage("TypeScript", []string{"//"}, [][]string{{"/*", "*/"}}),
			"HCL":                 NewLanguage("HCL", []string{"#", "//"}, [][]string{{"/*", "*/"}}),
			"Umka":                NewLanguage("Umka", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Unity-Prefab":        NewLanguage("Unity-Prefab", []string{}, [][]string{{"", ""}}),
			"MSBuild script":      NewLanguage("MSBuild script", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"Vala":                NewLanguage("Vala", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Verilog":             NewLanguage("Verilog", []string{"//"}, [][]string{{"/*", "*/"}}),
			"VimL":                NewLanguage("VimL", []string{`"`}, [][]string{{"", ""}}),
			"Visual Basic":        NewLanguage("Visual Basic", []string{"'"}, [][]string{{"", ""}}),
			"Vue":                 NewLanguage("Vue", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"Vyper":               NewLanguage("Vyper", []string{"#"}, [][]string{{"\"\"\"", "\"\"\""}}),
			"WiX":                 NewLanguage("WiX", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"XML":                 NewLanguage("XML", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"XML resource":        NewLanguage("XML resource", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"XSLT":                NewLanguage("XSLT", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"XSD":                 NewLanguage("XSD", []string{"<!--"}, [][]string{{"<!--", "-->"}}),
			"YAML":                NewLanguage("YAML", []string{"#"}, [][]string{{"", ""}}),
			"Yacc":                NewLanguage("Yacc", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Yul":                 NewLanguage("Yul", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Zephir":              NewLanguage("Zephir", []string{"//"}, [][]string{{"/*", "*/"}}),
			"Zig":                 NewLanguage("Zig", []string{"//", "///"}, [][]string{{"", ""}}),
			"Zsh":                 NewLanguage("Zsh", []string{"#"}, [][]string{{"", ""}}),
		},
	}
}
