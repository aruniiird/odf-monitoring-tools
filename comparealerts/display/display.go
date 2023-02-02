package display

import "comparealerts/types"

type AlertDiffDisplayer interface {
	Display(diffUnits []types.PrometheusRuleDiffUnit)
}
