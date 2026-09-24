package present

import (
	"errors"
	"testing"
)

type fakeHRESULT uintptr

func (f fakeHRESULT) Error() string { return "hresult" }
func (f fakeHRESULT) Code() uintptr { return uintptr(f) }

func TestIgnoreAlreadyInitialized(t *testing.T) {
	cases := []struct {
		name string
		in   error
		ok   bool
	}{
		{"nil", nil, true},
		{"S_FALSE", fakeHRESULT(hresultSFalse), true},
		{"RPC_E_CHANGED_MODE", fakeHRESULT(hresultRPCChangedMode), true},
		{"E_FAIL", fakeHRESULT(0x80004005), false},
		{"plain error", errors.New("boom"), false},
	}
	for _, c := range cases {
		if got := ignoreAlreadyInitialized(c.in); (got == nil) != c.ok {
			t.Errorf("%s: ignoreAlreadyInitialized() = %v, want ok=%v", c.name, got, c.ok)
		}
	}
}
