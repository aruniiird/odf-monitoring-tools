package types_test

import (
	"testing"

	"comparealerts/types"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func newTypesIntOrString(str string) types.IntOrString {
	return types.IntOrString{
		IntOrString: intstr.IntOrString{
			Type: intstr.String, StrVal: str,
		},
	}
}

func TestIntOrString(t *testing.T) {
	testStrings := []string{
		"ab  c",
		`ab
		c
		`,
		"a\tb\n\nc d",
		"ab \"\tc      \", d   ,\n\ne  , f\n",
	}
	expectedStrings := []string{
		"ab c",
		"ab c",
		"a b c d",
		`ab " c ", d , e , f`,
	}
	for i, eachTestStr := range testStrings {
		iOrS := newTypesIntOrString(eachTestStr)
		if actualResult := iOrS.TrimWithOnlySpaces(); actualResult != expectedStrings[i] {
			t.Errorf("String: %q \tExpected: %q \tActual: %q",
				eachTestStr, expectedStrings[i], actualResult)
			t.FailNow()
		}
	}
}
