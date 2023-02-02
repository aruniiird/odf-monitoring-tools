package main

import (
	"comparealerts/display"
	"comparealerts/display/prettyprint"
	"comparealerts/types"
	"errors"
	"flag"
	"io"
	"log"
	"os"
)

type MainOpts struct {
	verbose      bool
	fileArg1     string
	fileArg2     string
	outputFormat string
	outputFile   string
	flagSet      *flag.FlagSet
}

func NewMainOpts() *MainOpts {
	mainOpts := new(MainOpts)
	mainOpts.flagSet = flag.NewFlagSet("Alert diff tool", flag.ExitOnError)
	mainOpts.flagSet.BoolVar(&mainOpts.verbose, "verbose", false, "for verbose output")
	mainOpts.flagSet.BoolVar(&mainOpts.verbose, "v", false, "")
	mainOpts.flagSet.StringVar(&mainOpts.outputFormat, "output-format", "pp", "output format, expected values are [pp | pretty-print]")
	mainOpts.flagSet.StringVar(&mainOpts.outputFormat, "of", "pp", "")
	mainOpts.flagSet.StringVar(&mainOpts.outputFile, "output-file", "", "output can be stored to the file")
	mainOpts.flagSet.StringVar(&mainOpts.outputFile, "o", "", "")
	mainOpts.flagSet.Usage = func() {
		defaultLog := log.Default()
		defaultLog.SetFlags(0)
		defaultLog.Printf("%s [-v|-verbose] [-output-format|-of=pp|pretty-print] [-h|-help] alertFile1 alertFile2\n", os.Args[0])
		defaultLog.Println(`
-h  | -help
		prints this help message and exit
-of | -output-format <pp | pretty-print>
		output format, expected values are [pp | pretty-print] (default "pp")
-o  | -output-file   <output-file-name>
		output file arg, if provided will write the output to the file as well
-v  | -verbose
		for verbose output`)
	}
	return mainOpts
}

func (mainOpts *MainOpts) ParseArgs(args []string) error {
	if mainOpts == nil {
		return errors.New("MainOpts object should not be 'nil'")
	}
	if mainOpts.flagSet.Parsed() {
		return errors.New("MainOpts flagset is already parsed")
	}
	if err := mainOpts.flagSet.Parse(args); err != nil {
		return err
	}
	args = mainOpts.flagSet.Args()
	if len(args) < 2 {
		return errors.New("requires two alert files to compare")
	}
	mainOpts.fileArg1 = args[0]
	mainOpts.fileArg2 = args[1]
	switch mainOpts.outputFormat {
	case "pp", "pretty-print":
		break // above values are all accepted
	default:
		return errors.New("unsupported output format: " + mainOpts.outputFormat)
	}
	return nil
}

func (mainOpts *MainOpts) ParseDefaultArgs() error {
	return mainOpts.ParseArgs(os.Args[1:])
}

func main() {
	logFlag := log.LstdFlags | log.Lshortfile
	outLog := log.New(os.Stdout, "", log.LstdFlags)
	errLog := log.New(os.Stderr, "[ERROR] ", logFlag)
	verboseLog := log.New(io.Discard, "[INFO] ", logFlag)
	opts := NewMainOpts()
	if err := opts.ParseDefaultArgs(); err != nil {
		errLog.Println("Error: ", err)
		opts.flagSet.Usage()
		os.Exit(1)
	}

	if opts.verbose {
		verboseLog.SetOutput(os.Stderr)
	}

	if opts.outputFile != "" {
		if f, err := os.Create(opts.outputFile); err == nil {
			defer f.Close()
			outLog.SetOutput(io.MultiWriter(f, outLog.Writer()))
			errLog.SetOutput(io.MultiWriter(f, errLog.Writer()))
			if opts.verbose {
				verboseLog.SetOutput(io.MultiWriter(f, verboseLog.Writer()))
			}
		}
	}

	var displayMode display.AlertDiffDisplayer
	switch opts.outputFormat {
	case "pp", "pretty-print":
		displayMode = prettyprint.NewDiffUnitsPrettyPrinter(outLog)
	}

	verboseLog.Println("Compare files:")
	verboseLog.Println("File 1 > ", opts.fileArg1)
	verboseLog.Println("File 2 > ", opts.fileArg2)
	promRuleOne, err := types.NewPrometheusRule(
		types.NewPromRuleOptions(opts.fileArg1, verboseLog))
	if err != nil {
		errLog.Fatalf("Alert file1: %q error: %v", opts.fileArg1, err)
	}
	promRuleTwo, err := types.NewPrometheusRule(
		types.NewPromRuleOptions(opts.fileArg2, verboseLog))
	if err != nil {
		errLog.Fatalf("Alert file2: %q error: %v", opts.fileArg2, err)
	}
	displayMode.Display(promRuleOne.Diff(promRuleTwo))
}
