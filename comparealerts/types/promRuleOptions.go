package types

import (
	"io"
	"log"
)

type PromRuleOptions struct {
	alertFile string
	outLog    *log.Logger
}

func NewPromRuleOptions(alertFile string, outLog *log.Logger) *PromRuleOptions {
	if outLog == nil {
		outLog = log.New(io.Discard, "", log.LstdFlags)
	}
	promRuleOpts := &PromRuleOptions{
		alertFile: alertFile, outLog: outLog,
	}
	return promRuleOpts
}
