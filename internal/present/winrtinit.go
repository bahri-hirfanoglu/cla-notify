package present

import "errors"

const (
	hresultSFalse         = 0x00000001
	hresultRPCChangedMode = 0x80010106
)

type hresultError interface{ Code() uintptr }

// ignoreAlreadyInitialized treats "already initialized" results of COM/WinRT startup as success.
func ignoreAlreadyInitialized(err error) error {
	var he hresultError
	if errors.As(err, &he) {
		switch uint32(he.Code()) {
		case hresultSFalse, hresultRPCChangedMode:
			return nil
		}
	}
	return err
}
