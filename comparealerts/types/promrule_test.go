package types_test

import (
	"comparealerts/types"
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
	"testing"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	yamlContent1_sha256 = [sha256.Size]byte{
		236, 128, 229, 111, 16, 116, 128, 254,
		13, 61, 36, 133, 51, 174, 128, 193,
		37, 52, 243, 255, 199, 64, 199, 27,
		166, 40, 241, 136, 131, 214, 142, 69}
	yamlContent2_sha256 = [sha256.Size]byte{
		50, 218, 70, 55, 125, 182, 63, 40,
		71, 211, 93, 63, 82, 57, 95, 181,
		159, 235, 85, 156, 118, 229, 146, 221,
		254, 25, 155, 137, 186, 141, 227, 44}
)

var (
	expectedDiffs1 = []types.PrometheusRuleDiffUnit{
		{Alert: "OdfMirrorDaemonStatus2", MainReason: types.DiffReasonSubUnit{
			DiffReason: types.DifferentExpr, DiffMessage: types.DifferentExpr.String(),
			Rule1: []types.Rule{{Alert: "OdfMirrorDaemonStatus2",
				Expr: types.IntOrString{IntOrString: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: `((count by(namespace) (ocs_mirror_daemon_count{job="ocs-metrics-exporter"} == 0)) * on(namespace) group_left() (count by(namespace) (ocs_pool_mirroring_status{job="ocs-metrics-exporter"} == 1))) > 0`}, LineNo: 19}, For: "1m", Labels: map[string]string{"severity": "critical"},
				Annotations: map[string]string{
					"description":    "Mirror daemon is in unhealthy status for more than 1m. Mirroring on this cluster is not working as expected.",
					"message":        "Mirror daemon is unhealthy.",
					"severity_level": "error", "storage_type": "ceph"}}},
			Rule2: []types.Rule{{Alert: "OdfMirrorDaemonStatus2",
				Expr: types.IntOrString{IntOrString: intstr.IntOrString{Type: intstr.String, StrVal: `((count by(namespace) (ocs_mirror_daemon_count{job="ocs-metrics-exporter"} == 0)) * on(namespace) group_left() (count by(namespace) (ocs_pool_mirroring_status{job="ocs-metrics-exporter"} == 1))) > 3`}, LineNo: 19}, For: "1m", Labels: map[string]string{"severity": "critical"}, Annotations: map[string]string{"description": "Mirror daemon is in unhealthy status for more than 1m. Mirroring on this cluster is not working as expected.", "message": "Mirror daemon is unhealthy.", "severity_level": "error", "storage_type": "ceph"}}}},
		},
		{Alert: "OdfPoolMirroringImageHealth", MainReason: types.DiffReasonSubUnit{
			DiffReason:  types.AlertOnlyWithMe,
			DiffMessage: types.AlertOnlyWithMe.String(),
			Rule1: []types.Rule{{
				Alert: "OdfPoolMirroringImageHealth",
				Expr:  types.IntOrString{IntOrString: intstr.IntOrString{Type: intstr.String, StrVal: `(ocs_pool_mirroring_image_health{job="ocs-metrics-exporter"}  * on (namespace) group_left() (max by(namespace) (ocs_pool_mirroring_status{job="ocs-metrics-exporter"}))) == 1`}, LineNo: 41},
				For:   "1m", Labels: map[string]string{"severity": "warning"},
				Annotations: map[string]string{"description": "Mirroring image(s) (PV) in the pool {{ $labels.name }} are in Unknown state for more than 1m. Mirroring might not work as expected.", "message": "Mirroring image(s) (PV) in the pool {{ $labels.name }} are in Unknown state.", "severity_level": "warning", "storage_type": "ceph"}}},
			Rule2: []types.Rule{}}},
	}
	expectedDiffs2 = []types.PrometheusRuleDiffUnit{
		{
			Alert: "CephMgrIsMissingReplicas",
			MainReason: types.DiffReasonSubUnit{
				DiffReason:  types.DifferentExpr,
				DiffMessage: "difference in expression",
				Rule1: []types.Rule{
					{
						Alert: "CephMgrIsMissingReplicas",
						Expr: types.IntOrString{
							IntOrString: intstr.IntOrString{Type: intstr.String, StrVal: `sum(kube_deployment_spec_replicas{deployment=~"rook-ceph-mgr-.*"}) by (namespace) < 1`},
							LineNo:      1,
						},
						For:         "5m",
						Labels:      map[string]string{"severity": "warning"},
						Annotations: map[string]string{"description": "Ceph Manager is missing replicas.", "message": "Storage metrics collector service doesn't have required no of replicas.", "severity_level": "warning", "storage_type": "ceph"},
					},
				},
				Rule2: []types.Rule{
					{
						Alert: "CephMgrIsMissingReplicas",
						Expr: types.IntOrString{
							IntOrString: intstr.IntOrString{Type: intstr.String, StrVal: `ABC > 10`},
							LineNo:      1,
						},
						For:         "5m",
						Labels:      map[string]string{"severity": "warning"},
						Annotations: map[string]string{"description": "Ceph Manager is missing replicas.", "message": "Storage metrics collector service doesn't have required no of replicas.", "severity_level": "warning", "storage_type": "ceph"},
					},
				},
			},
		},
	}
)

func testDataIntegrity(t *testing.T) {
	var testData = []string{yamlContent1, yamlContent2}
	var expectedSha256 = [][sha256.Size]byte{
		yamlContent1_sha256, yamlContent2_sha256}
	for idx, tData := range testData {
		if actualSha256 := sha256.Sum256([]byte(tData)); expectedSha256[idx] != actualSha256 {
			t.Errorf("Expected: %+v Actual: %+v", expectedSha256[idx], actualSha256)
			t.Errorf("test content%d changed", idx+1)
			t.FailNow()
		}
	}
}

func TestNewPrometheusRuleFromFile(t *testing.T) {
	_, err := types.NewPrometheusRule(types.NewPromRuleOptions("non-existing-file.txt", nil))
	if err == nil {
		t.Errorf("Function 'NewPrometheusRuleFromFile()' is supposed to throw an error")
		t.FailNow()
	}
	testDataIntegrity(t)

	prObj1 := promRuleObj.DeepCopy()
	jsonStr1 := jsonDataFromMonPrometheusRule(t, prObj1)
	prObj2 := promRuleObj.DeepCopy()
	changeAnAlertExpr("CephMgrIsMissingReplicas", "ABC > 10", prObj2)
	jsonStr2 := jsonDataFromMonPrometheusRule(t, prObj2)

	testDataArr := []struct {
		fileContent1, fileContent2 *string
		fileType                   string
		expecedDiff                []types.PrometheusRuleDiffUnit
	}{
		{fileContent1: &yamlContent1, fileContent2: &yamlContent2,
			fileType: "yaml", expecedDiff: expectedDiffs1},
		{fileContent1: &jsonStr1, fileContent2: &jsonStr2, fileType: "json", expecedDiff: expectedDiffs2},
	}
	for _, td := range testDataArr {
		tmpFile1 := createATempFile(t, "tmpPromRule", td.fileType, td.fileContent1)
		tmpFile2 := createATempFile(t, "tmpPromRule", td.fileType, td.fileContent2)
		defer func() { os.Remove(tmpFile1); os.Remove(tmpFile2) }()
		promRule1, err := types.NewPrometheusRule(types.NewPromRuleOptions(tmpFile1, nil))
		if err != nil {
			t.Errorf("Failed to create a prometheus rule object from file: %s", tmpFile1)
			t.Errorf("Error: %v", err)
			t.FailNow()
		}
		promRule2, err := types.NewPrometheusRule(types.NewPromRuleOptions(tmpFile2, nil))
		if err != nil {
			t.Errorf("Failed to create a prometheus rule object from file: %s", tmpFile2)
			t.Errorf("Error: %v", err)
			t.FailNow()
		}
		diffs := promRule1.Diff(promRule2)
		compareDiffs(t, diffs, td.expecedDiff)
	}
}

func createATempFile(t *testing.T, prefix, ext string, content *string) string {
	if content == nil {
		t.Errorf("We don't have any contents")
		t.FailNow()
	}
	f1, err := os.CreateTemp("", fmt.Sprintf("%s*.%s", prefix, ext))
	if err != nil {
		t.Error("Creating a temporary file failed")
		t.FailNow()
	}
	defer f1.Close()
	if _, err := f1.WriteString(*content); err != nil {
		t.Errorf("Writing to the tmp file failed: %s", f1.Name())
		os.Remove(f1.Name())
		t.FailNow()
	}
	return f1.Name()
}

func compareDiffs(t *testing.T, diffs1, diffs2 []types.PrometheusRuleDiffUnit) {
	if diff1Len, diff2Len := len(diffs1), len(diffs2); diff1Len != diff2Len {
		t.Errorf("Length of the diffs don't match. Diff1Len: %d Diff2Len: %d",
			diff1Len, diff2Len)
		t.FailNow()
	}
	for _, diff1 := range diffs1 {
		alertNameNotFound := true
		for _, diff2 := range diffs2 {
			if diff1.Alert != diff2.Alert {
				continue
			}
			diff1Reason := diff1.MainReason
			alertNameNotFound = false
			diff2Reason := diff2.MainReason
			if diff1Reason.DiffReason != diff2Reason.DiffReason {
				t.Errorf("Diff reasons don't match: R1: %v R2: %v",
					diff1Reason.DiffReason, diff2Reason.DiffReason)
				t.FailNow()
			}
			if len(diff1Reason.Rule1) != len(diff2Reason.Rule1) {
				t.Errorf("No: of rules in RuleSet-1 don't match")
				t.FailNow()
			}
			if len(diff1Reason.Rule2) != len(diff2Reason.Rule2) {
				t.Errorf("No: of rules in RuleSet-2 don't match")
				t.FailNow()
			}
			for ruleIndx, diff1Rule1 := range diff1Reason.Rule1 {
				diff2Rule1 := diff2Reason.Rule1[ruleIndx]
				expr1 := diff1Rule1.Expr.TrimWithOnlySpaces()
				expr2 := diff2Rule1.Expr.TrimWithOnlySpaces()
				if expr1 != expr2 {
					t.Errorf("RuleSet-1 expressions don't match:\nExpr1: %s\nExpr2: %s", expr1, expr2)
					t.FailNow()
				}
			}
			for ruleIndx, diff1Rule2 := range diff1Reason.Rule2 {
				diff2Rule2 := diff2Reason.Rule2[ruleIndx]
				expr1 := diff1Rule2.Expr.TrimWithOnlySpaces()
				expr2 := diff2Rule2.Expr.TrimWithOnlySpaces()
				if expr1 != expr2 {
					t.Errorf("RuleSet-2 expressions don't match:\nExpr1: %s\nExpr2: %s", expr1, expr2)
					t.FailNow()
				}
			}
		}
		if alertNameNotFound {
			t.Errorf("Alert name: %s not found", diff1.Alert)
			t.FailNow()
		}
	}
}

func TestUniqueDiffs(t *testing.T) {
	testData := []struct {
		mapA          map[string][]types.Rule
		mapB          map[string][]types.Rule
		expectedDiffs []types.PrometheusRuleDiffUnit
	}{
		{
			mapA: map[string][]types.Rule{"a": nil, "b": nil, "c": nil},
			mapB: map[string][]types.Rule{"e": nil, "b": nil, "d": nil},
			expectedDiffs: []types.PrometheusRuleDiffUnit{
				{
					Alert:      "a",
					MainReason: types.DiffReasonSubUnit{DiffReason: types.AlertOnlyWithMe},
				},
				{
					Alert:      "c",
					MainReason: types.DiffReasonSubUnit{DiffReason: types.AlertOnlyWithMe},
				},
				{
					Alert:      "e",
					MainReason: types.DiffReasonSubUnit{DiffReason: types.AlertOnlyWithThem},
				},
				{
					Alert:      "d",
					MainReason: types.DiffReasonSubUnit{DiffReason: types.AlertOnlyWithThem},
				},
			},
		},
		{
			mapA: map[string][]types.Rule{"a": nil, "b": nil, "c": nil},
			mapB: map[string][]types.Rule{"a": nil, "c": nil, "d": nil},
			expectedDiffs: []types.PrometheusRuleDiffUnit{
				{
					Alert:      "b",
					MainReason: types.DiffReasonSubUnit{DiffReason: types.AlertOnlyWithMe},
				},
				{
					Alert:      "d",
					MainReason: types.DiffReasonSubUnit{DiffReason: types.AlertOnlyWithThem},
				},
			},
		},
	}
	for _, td := range testData {
		diffs := types.UniqueDiffs(td.mapA, td.mapB)
		for _, diff := range diffs {
			diffMatched := false
			for _, expectedDiff := range td.expectedDiffs {
				if diff.Alert == expectedDiff.Alert &&
					diff.MainReason.DiffReason == expectedDiff.MainReason.DiffReason {
					diffMatched = true
					break
				}
			}
			if !diffMatched {
				t.Errorf("No match found for alert: %q", diff.Alert)
				t.FailNow()
			}
		}
	}
}

func genFunc(startKeyNum, noOfKeys int) map[string][]types.Rule {
	retMap := make(map[string][]types.Rule, noOfKeys)
	for i := startKeyNum; i < startKeyNum+noOfKeys; i++ {
		retMap[strconv.Itoa(i)] = nil
	}
	return retMap
}

func BenchmarkUniqueDiffs(b *testing.B) {
	benchmarkUniqueDiffs(b, types.UniqueDiffs)
}

func benchmarkUniqueDiffs(b *testing.B, uniqueFunc func(map1, map2 map[string][]types.Rule) []types.PrometheusRuleDiffUnit) {
	benchN := 10000
	mapA := genFunc(1, benchN)
	mapB := genFunc(11, benchN)
	for i := 0; i < b.N; i++ {
		var _ = uniqueFunc(mapA, mapB)
	}
}

func generateDummyLocalPrometheusRuleObj(startAlertCount, noOfAlerts int, promR *monv1.PrometheusRule) *types.PrometheusRule {
	promR1 := promR.DeepCopy()
	for i := startAlertCount; i < startAlertCount+noOfAlerts; i++ {
		addAnAlertToMonPromRule(monv1.Rule{Alert: "MyNewAlert" + strconv.Itoa(i),
			Expr: intstr.IntOrString{Type: intstr.String, StrVal: "abc > 10"}}, promR1)
	}
	return types.ConvertToLocalPrometheusRule(promR1)
}

func sequentialDiff(promRule1, promRule2 *types.PrometheusRule) []types.PrometheusRuleDiffUnit {
	prom1Alerts := promRule1.Alerts()
	prom2Alerts := promRule2.Alerts()
	var diffs []types.PrometheusRuleDiffUnit
	for alert1Name, alert1Rules := range prom1Alerts {
		var diffsInAlert1Name = types.PrometheusRuleDiffUnit{Alert: alert1Name}
		if alert2Rules, ok := prom2Alerts[alert1Name]; !ok {
			// first check: see if the alert is only with the first 'PrometheusRule' object
			diffsInAlert1Name.MainReason = types.NewDiffReasonSubUnit(types.AlertOnlyWithMe,
				"Alert only with file 1", alert1Rules, nil)
		} else if len(alert1Rules) != len(alert2Rules) {
			// second check: whether the rules array have different length
			diffsInAlert1Name.MainReason = types.NewDiffReasonSubUnit(types.DifferenceInNoOfRules,
				"No: of rules for the alert differs", alert1Rules, alert2Rules)
		} else {
			// third check:
			// we established: (a) both have the same alert with (b) same #:of rules
			// now check: those rules have same expressions or not
			var alert1RuleExprSame bool
			var differedAlert1Rules []types.Rule
			var differedAlert2Rules []types.Rule
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
					differedAlert1Rules = append(differedAlert1Rules, alert1Rules[alert1Indx])
					differedAlert2Rules = append(differedAlert2Rules, alert2Rules[alert1Indx])
				}
			}
			if len(differedAlert1Rules) > 0 {
				diffsInAlert1Name.MainReason = types.NewDiffReasonSubUnit(
					types.DifferentExpr, types.DifferentExpr.String(),
					differedAlert1Rules, differedAlert2Rules)
			}

		}
		if diffsInAlert1Name.MainReason.DiffReason > 0 {
			diffs = append(diffs, diffsInAlert1Name)
		}
	}
	for alert2Name := range prom2Alerts {
		var diffsInAlert2Name = types.PrometheusRuleDiffUnit{Alert: alert2Name}
		if _, ok := prom1Alerts[alert2Name]; !ok {
			diffsInAlert2Name.MainReason = types.NewDiffReasonSubUnit(types.AlertOnlyWithThem,
				"Alerts only with file2",
				nil, prom2Alerts[alert2Name])
			diffs = append(diffs, diffsInAlert2Name)
			continue
		}
	}
	return diffs
}

func BenchmarkSequentialDiff(b *testing.B) {
	benchmarkDiffs(b, sequentialDiff)
}

func BenchmarkParallelDiff(b *testing.B) {
	parallelDiff := func(promRule1, promRule2 *types.PrometheusRule) []types.PrometheusRuleDiffUnit {
		return promRule1.Diff(promRule2)
	}
	benchmarkDiffs(b, parallelDiff)
}

func benchmarkDiffs(b *testing.B, diffFunc func(promRule1, promRule2 *types.PrometheusRule) []types.PrometheusRuleDiffUnit) {
	promR1 := promRuleObj.DeepCopy()
	promR2 := promRuleObj.DeepCopy()
	changeAnAlertExpr("CephMgrIsMissingReplicas", "ABC > 10", promR2)
	promLocalR1 := generateDummyLocalPrometheusRuleObj(0, 100, promR1)
	promLocalR2 := generateDummyLocalPrometheusRuleObj(20, 100, promR2)
	for i := 0; i < b.N; i++ {
		var _ = diffFunc(promLocalR1, promLocalR2)
	}
}
