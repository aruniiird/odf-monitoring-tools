package prettyprint

import (
	"log"

	"comparealerts/display"
	"comparealerts/types"
)

type DiffUnitsPrettyPrinter struct {
	logger *log.Logger
}

func NewDiffUnitsPrettyPrinter(logger *log.Logger) *DiffUnitsPrettyPrinter {
	return &DiffUnitsPrettyPrinter{logger: logger}
}

var _ display.AlertDiffDisplayer = &DiffUnitsPrettyPrinter{}

func (pp *DiffUnitsPrettyPrinter) Display(diffUnits []types.PrometheusRuleDiffUnit) {
	pp.logger.Println("--")
	if len(diffUnits) == 0 {
		pp.logger.Println("No diffs found")
		pp.logger.Println("--")
	}
	for diffIndx, eachDiffUnit := range diffUnits {
		pp.logger.Printf("Alert #%02d: %s\n", diffIndx+1, eachDiffUnit.Alert)
		pp.logger.Println("Reasons  : ")
		mainReason := &eachDiffUnit.MainReason
		pp.logger.Println("  Reason : ", mainReason.DiffReason.Name())
		pp.logger.Println("  Message: ", mainReason.DiffMessage)
		switch mainReason.DiffReason {
		case types.DifferenceInNoOfRules:
			pp.logger.Printf("  Alert File 1 #Rules: %d", len(mainReason.Rule1))
			pp.logger.Printf("  Alert File 2 #Rules: %d", len(mainReason.Rule2))
		case types.DifferentExpr:
			for indx := range mainReason.Rule1 {
				rule1 := mainReason.Rule1[indx]
				rule2 := mainReason.Rule2[indx]
				pp.logger.Printf("  Expression 1: %s", rule1.Expr.IntOrString.String())
				pp.logger.Printf("  Expression 2: %s", rule2.Expr.IntOrString.String())
			}
		}
		pp.logger.Println("--")
	}
}
