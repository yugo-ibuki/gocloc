package main

import (
	"fmt"
	"github.com/hhatto/gocloc"
	flags "github.com/jessevdk/go-flags"
	"sort"
)

const languageHeader string = "Language"
const commonHeader string = "files          blank        comment           code"
const defaultOutputSeparator string = "-------------------------------------------------------------------------" +
	"-------------------------------------------------------------------------" +
	"-------------------------------------------------------------------------"

var rowLen = 79

// CmdOptions is gocloc command options.
// It is necessary to use notation that follows go-flags.
type CmdOptions struct {
	Byfile   bool   `long:"by-file" description:"report results for every encountered source file"`
	MatchDir string `long:"match-d" description:"include dir name (regex)"`
}

type outputBuilder struct {
	opts   *CmdOptions
	result *gocloc.Result
}

func newOutputBuilder(result *gocloc.Result, opts *CmdOptions) *outputBuilder {
	return &outputBuilder{
		opts,
		result,
	}
}

func (o *outputBuilder) WriteHeader() {
	maxPathLen := o.result.MaxPathLength
	headerLen := 28
	header := languageHeader
	rowLen = maxPathLen + len(commonHeader) + 2
	fmt.Printf("%.[2]*[1]s\n", defaultOutputSeparator, rowLen)
	fmt.Printf("%-[2]*[1]s %[3]s\n", header, headerLen, commonHeader)
	fmt.Printf("%.[2]*[1]s\n", defaultOutputSeparator, rowLen)
}

func (o *outputBuilder) WriteFooter() {
	total := o.result.Total
	maxPathLen := o.result.MaxPathLength

	fmt.Printf("%.[2]*[1]s\n", defaultOutputSeparator, rowLen)
	if o.opts.Byfile {
		fmt.Printf("%-[1]*[2]v %6[3]v %14[4]v %14[5]v %14[6]v\n",
			maxPathLen, "TOTAL", total.Total, total.Blanks, total.Comments, total.Code)
	} else {
		fmt.Printf("%-27v %6v %14v %14v %14v\n",
			"TOTAL", total.Total, total.Blanks, total.Comments, total.Code)
	}
	fmt.Printf("%.[2]*[1]s\n", defaultOutputSeparator, rowLen)
}

func (o *outputBuilder) WriteResult() {
	o.WriteHeader()

	clocLangs := o.result.Languages

	var sortedLanguages gocloc.Languages
	for _, language := range clocLangs {
		if len(language.Files) != 0 {
			sortedLanguages = append(sortedLanguages, *language)
		}
	}
	sort.Sort(sortedLanguages)

	for _, language := range sortedLanguages {
		fmt.Printf("%-27v %6v %14v %14v %14v\n",
			language.Name, len(language.Files), language.Blanks, language.Comments, language.Code)
	}

	o.WriteFooter()
}

func main() {
	var opts CmdOptions
	clocOpts := gocloc.NewClocOptions()
	// parse command line options
	parser := flags.NewParser(&opts, flags.Default)
	parser.Name = "gocloc"
	parser.Usage = "[OPTIONS] PATH[...]"

	paths, err := flags.Parse(&opts)
	if err != nil {
		return
	}

	// value for language result
	languages := gocloc.NewDefinedLanguages()

	processor := gocloc.NewProcessor(languages, clocOpts)
	result, err := processor.Analyze(paths)
	if err != nil {
		fmt.Printf("fail gocloc analyze. error: %v\n", err)
		return
	}

	builder := newOutputBuilder(result, &opts)
	builder.WriteResult()
}
