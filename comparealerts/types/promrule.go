package types

import (
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type PrometheusRule struct {
	Spec PrometheusRuleSpec `yaml:"spec,omitempty"`
	// some times the file can directly have 'PrometheusRuleSpec' fields
	PrometheusRuleSpec `yaml:"groups,omitempty"`
	prOpts             *PromRuleOptions
}
type PrometheusRuleSpec struct {
	Groups []RuleGroup `yaml:"groups,omitempty"`
}
type RuleGroup struct {
	Name                    string `yaml:"name"`
	Interval                string `yaml:"interval,omitempty"`
	Rules                   []Rule `yaml:"rules"`
	PartialResponseStrategy string `yaml:"partial_response_strategy,omitempty"`
}
type Rule struct {
	Record      string            `yaml:"record,omitempty"`
	Alert       string            `yaml:"alert,omitempty"`
	Expr        IntOrString       `yaml:"expr"`
	For         string            `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

func NewPrometheusRule(prOpts *PromRuleOptions) (*PrometheusRule, error) {
	if prOpts == nil {
		return nil, errors.New("'nil' Prometheus Rule Options provided")
	}
	promRuleFile, err := os.Open(prOpts.alertFile)
	if err != nil {
		return nil, err
	}
	defer promRuleFile.Close()
	var promRule *PrometheusRule
	promRule, err = ParseFromYaml(promRuleFile)
	if err != nil {
		promRule, err = ParseFromJson(promRuleFile)
	}
	if err != nil {
		err = errors.New("corrupted yaml or json file: " + prOpts.alertFile)
		return nil, err
	}
	promRule.prOpts = prOpts
	// if `promRule.groups` part is populated directly
	// add it to the promRule.spec.groups (used for all the diff calculations)
	// and clear out the groups part
	if len(promRule.Groups) > 0 {
		// raise an error if 'spec.groups' is also populated
		if len(promRule.Spec.Groups) > 0 {
			return nil, errors.New("malformated file, both 'spec.groups' and 'groups' should not be populated")
		}
		promRule.Spec.Groups = promRule.Groups
		promRule.Groups = nil
	}
	return promRule, nil
}

func (promRule *PrometheusRule) Opts() *PromRuleOptions {
	return promRule.prOpts
}

// Alerts function collects all the alerts into a map
func (promRule *PrometheusRule) Alerts() map[string][]Rule {
	if promRule == nil {
		return nil
	}
	allAlerts := make(map[string][]Rule)
	for _, eachRuleGroup := range promRule.Spec.Groups {
	topRuleLabel:
		for _, topRule := range eachRuleGroup.Rules {
			if topRule.Alert == "" {
				continue
			}
			if collectedRules, ok := allAlerts[topRule.Alert]; ok {
				for _, collectedRule := range collectedRules {
					if collectedRule.Expr.TrimWithOnlySpaces() == topRule.Expr.TrimWithOnlySpaces() {
						continue topRuleLabel
					}
				}
				promRule.prOpts.outLog.Printf("Multiple entries found for alert: %q", topRule.Alert)
			}
			allAlerts[topRule.Alert] = append(allAlerts[topRule.Alert], topRule)
		}
	}
	return allAlerts
}

func (promRule *PrometheusRule) Diff(promRule2 *PrometheusRule) []PrometheusRuleDiffUnit {
	prom1Alerts := promRule.Alerts()
	prom2Alerts := promRule2.Alerts()
	var diffs []PrometheusRuleDiffUnit
	// get the unique alerts from both the files
	diffs = UniqueDiffs(prom1Alerts, prom2Alerts)
	// remove those unique alert names from both the maps
	for _, diff := range diffs {
		delete(prom1Alerts, diff.Alert)
		delete(prom2Alerts, diff.Alert)
	}
	for alert1Name, alert1Rules := range prom1Alerts {
		diffUnit := PrometheusRuleDiffUnit{Alert: alert1Name}
		alert2Rules := prom2Alerts[alert1Name]
		// if the alert's rules length differs
		if len(alert1Rules) != len(alert2Rules) {
			diffUnit.MainReason = NewDiffReasonSubUnit(DifferenceInNoOfRules,
				"No: of rules for the alert differs", alert1Rules, alert2Rules)
		} else {
			// or else check the expressions
			notMatchingIndices := diffInAlertExpressionIndices(alert1Rules, alert2Rules)
			var notMatchingRule1, notMatchingRule2 []Rule
			for _, notMatchingIndx := range notMatchingIndices {
				notMatchingRule1 = append(notMatchingRule1, alert1Rules[notMatchingIndx])
				notMatchingRule2 = append(notMatchingRule2, alert2Rules[notMatchingIndx])
			}
			if len(notMatchingIndices) > 0 {
				diffUnit.MainReason = NewDiffReasonSubUnit(DifferentExpr,
					"Rule expressions don't match",
					notMatchingRule1, notMatchingRule2)
			}

		}
		if diffUnit.MainReason.DiffReason > 0 {
			diffs = append(diffs, diffUnit)
		}
	}
	return diffs
}

func UniqueDiffs(map1, map2 map[string][]Rule) []PrometheusRuleDiffUnit {
	getKeysOnlyInMapA := func(mapA, mapB map[string][]Rule,
		basicDiffSU DiffReasonSubUnit) <-chan []PrometheusRuleDiffUnit {
		diffCh := make(chan []PrometheusRuleDiffUnit)
		go func(diffCh chan<- []PrometheusRuleDiffUnit) {
			defer close(diffCh)
			diffUnitArr := make([]PrometheusRuleDiffUnit, 0, 5)
			for alertNameKey := range mapA {
				if _, ok := mapB[alertNameKey]; !ok {
					rule1 := mapA[alertNameKey]
					var rule2 []Rule
					if basicDiffSU.DiffReason == AlertOnlyWithThem {
						rule1, rule2 = rule2, rule1
					}
					diffUnit := PrometheusRuleDiffUnit{Alert: alertNameKey}
					diffUnit.MainReason = DiffReasonSubUnit{
						DiffReason: basicDiffSU.DiffReason, DiffMessage: basicDiffSU.DiffMessage, Rule1: rule1, Rule2: rule2,
					}
					diffUnitArr = append(diffUnitArr, diffUnit)
				}
			}
			diffCh <- diffUnitArr
		}(diffCh)
		return diffCh
	}
	uniqueMap1KeysCh := getKeysOnlyInMapA(map1, map2, NewDiffReasonSubUnit(AlertOnlyWithMe, "Alert only with file 1", nil, nil))
	uniqueMap2KeysCh := getKeysOnlyInMapA(map2, map1, NewDiffReasonSubUnit(AlertOnlyWithThem, "Alert only with file 2", nil, nil))
	diffUnits := <-uniqueMap1KeysCh
	diffUnits = append(diffUnits, <-uniqueMap2KeysCh...)
	return diffUnits
}

func diffInAlertExpressionIndices(alert1Rules, alert2Rules []Rule) []int {
	var alert1RuleExprSame bool
	var notMatchingRuleIndices []int
	for alert1Indx, ruleInAlert1 := range alert1Rules {
		alert1RuleExprSame = false
		ruleInAlert1ExprTrimmed := ruleInAlert1.Expr.TrimWithOnlySpaces()
		for _, ruleInAlert2 := range alert2Rules {
			ruleInAlert2ExprTrimmed := ruleInAlert2.Expr.TrimWithOnlySpaces()
			if ruleInAlert1ExprTrimmed == ruleInAlert2ExprTrimmed {
				alert1RuleExprSame = true
				break
			}
		}
		// collect all the rules whose expressions differ
		if !alert1RuleExprSame {
			notMatchingRuleIndices = append(notMatchingRuleIndices, alert1Indx)
		}
	}
	return notMatchingRuleIndices
}

type IntOrString struct {
	intstr.IntOrString
	LineNo int
}

func (iOrS *IntOrString) UnmarshalYAML(value *yaml.Node) error {
	iOrS.LineNo = value.Line
	iOrS.IntOrString = intstr.Parse(value.Value)
	return nil
}

// TrimWithOnlySpaces function will return the expression string with only spaces
func (iOrS *IntOrString) TrimWithOnlySpaces() string {
	str := iOrS.IntOrString.String()
	strFields := strings.Fields(str)
	str = strings.Join(strFields, " ")
	return str
}
