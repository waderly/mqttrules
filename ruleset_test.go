package mqttrules

import "testing"

func TestSetParameter(t *testing.T) {
	for _, c := range []struct {
		key, value string
	}{
		{"/rules/TTL", "Testing"},
		{"/rules/TTL", ""},
		{"", ""},
	} {
		SetParameter(c.key, c.value)
		result := GetParameter(c.key)
		if result != c.value {
			t.Errorf("GetParameter(%q) == %q, want %q", c.key, result, c.value)
		}
	}
}

func TestReplaceParamsInString(t *testing.T) {
	SetParameter("test1", "value1")
	for _, c := range []struct {
		template, expected string
	}{
		{"", ""},
		{"$$", "$"},
		{"$test1$", "value1"},
		{"testing$test1$123", "testingvalue1123"},
		{"testing$test2$123", "testing123"},
	} {
		result := ReplaceParamsInString(c.template)
		if result != c.expected {
			t.Errorf("ReplaceParamsInString(%q); expected='%q', actual='%q'", c.template, c.expected, result)
		}
	}
}

func ExampleSetParameter() {
	SetParameter("/rules/TTL", "60")
}

func ExampleGetParameter() {
	GetParameter("/rules/TTL")

	//	result := GetParameter("/rules/TTL")

}