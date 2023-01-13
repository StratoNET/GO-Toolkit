package toolkit

import "testing"

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	rs := testTools.RandomString(10)
	if len(rs) != 10 {
		t.Error("incorrect length random string returned, should be 10 characters")
	}
}
