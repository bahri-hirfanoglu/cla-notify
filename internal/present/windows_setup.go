//go:build windows

package present

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows/registry"
)

// GUIDs for the classic (non-WinRT) shell COM interfaces used to create the Start Menu
// shortcut and stamp it with an AppUserModelID. CLSID_ShellLink, IShellLinkW, IPersistFile
// and IPropertyStore are long-stable, widely published Win32 COM identifiers.
var (
	clsidShellLink    = ole.NewGUID("00021401-0000-0000-C000-000000000046")
	iidIShellLinkW    = ole.NewGUID("000214F9-0000-0000-C000-000000000046")
	iidIPersistFile   = ole.NewGUID("0000010b-0000-0000-C000-000000000046")
	iidIPropertyStore = ole.NewGUID("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99")
)

// pkeyAppUserModelID is PKEY_AppUserModel_ID (System.AppUserModel.ID).
var pkeyAppUserModelID = propertyKey{fmtid: *ole.NewGUID("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"), pid: 5}

const vtLPWStr = 31 // VT_LPWSTR

type propertyKey struct {
	fmtid ole.GUID
	pid   uint32
}

// propVariantLPWStr mirrors the leading fields of a Win32 PROPVARIANT holding a VT_LPWSTR.
type propVariantLPWStr struct {
	vt  uint16
	_   [3]uint16
	ptr uintptr
}

// --- IShellLinkW ---

type shellLinkWVtbl struct {
	ole.IUnknownVtbl
	GetPath             uintptr
	GetIDList           uintptr
	SetIDList           uintptr
	GetDescription      uintptr
	SetDescription      uintptr
	GetWorkingDirectory uintptr
	SetWorkingDirectory uintptr
	GetArguments        uintptr
	SetArguments        uintptr
	GetHotKey           uintptr
	SetHotKey           uintptr
	GetShowCmd          uintptr
	SetShowCmd          uintptr
	GetIconLocation     uintptr
	SetIconLocation     uintptr
	SetRelativePath     uintptr
	Resolve             uintptr
	SetPath             uintptr
}

type shellLinkW struct{ ole.IUnknown }

func (v *shellLinkW) vtbl() *shellLinkWVtbl {
	return (*shellLinkWVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *shellLinkW) SetPath(path string) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	err = callMethod(v.vtbl().SetPath, uintptr(unsafe.Pointer(v)), uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
	return err
}

func (v *shellLinkW) SetArguments(args string) error {
	p, err := syscall.UTF16PtrFromString(args)
	if err != nil {
		return err
	}
	err = callMethod(v.vtbl().SetArguments, uintptr(unsafe.Pointer(v)), uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
	return err
}

// --- IPersistFile ---

type persistFileVtbl struct {
	ole.IUnknownVtbl
	GetClassID uintptr
	IsDirty    uintptr
	Load       uintptr
	Save       uintptr
}

type persistFile struct{ ole.IUnknown }

func (v *persistFile) vtbl() *persistFileVtbl {
	return (*persistFileVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *persistFile) Save(path string) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	err = callMethod(v.vtbl().Save, uintptr(unsafe.Pointer(v)), uintptr(unsafe.Pointer(p)), 1)
	runtime.KeepAlive(p)
	return err
}

// --- IPropertyStore ---

type propertyStoreVtbl struct {
	ole.IUnknownVtbl
	GetCount uintptr
	GetAt    uintptr
	GetValue uintptr
	SetValue uintptr
	Commit   uintptr
}

type propertyStore struct{ ole.IUnknown }

func (v *propertyStore) vtbl() *propertyStoreVtbl {
	return (*propertyStoreVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *propertyStore) SetString(key propertyKey, value string) error {
	p, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		return err
	}
	pv := propVariantLPWStr{vt: vtLPWStr, ptr: uintptr(unsafe.Pointer(p))}
	err = callMethod(v.vtbl().SetValue, uintptr(unsafe.Pointer(v)), uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&pv)))
	runtime.KeepAlive(p)
	return err
}

func (v *propertyStore) Commit() error {
	return callMethod(v.vtbl().Commit, uintptr(unsafe.Pointer(v)))
}

// ensureShortcut creates or overwrites the per-user Start Menu shortcut, stamped with
// cla-notify's AppUserModelID, so the toast notifier has an app identity to show under.
func ensureShortcut(exePath string) error {
	if err := ensureWinRT(); err != nil {
		return err
	}

	path := shortcutPath(os.Getenv)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	unk, err := ole.CreateInstance(clsidShellLink, iidIShellLinkW)
	if err != nil {
		return err
	}
	link := (*shellLinkW)(unsafe.Pointer(unk))
	defer link.Release()

	if err := link.SetPath(exePath); err != nil {
		return err
	}
	if err := link.SetArguments(""); err != nil {
		return err
	}

	var store *propertyStore
	if err := link.IUnknown.PutQueryInterface(iidIPropertyStore, &store); err != nil {
		return err
	}
	defer store.Release()
	if err := store.SetString(pkeyAppUserModelID, aumid); err != nil {
		return err
	}
	if err := store.Commit(); err != nil {
		return err
	}

	var pf *persistFile
	if err := link.IUnknown.PutQueryInterface(iidIPersistFile, &pf); err != nil {
		return err
	}
	defer pf.Release()
	return pf.Save(path)
}

// ensureProtocolRegistration points the cla-notify:// URL protocol at exePath, per user,
// no admin rights required.
func ensureProtocolRegistration(exePath string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, protocolKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}

	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, protocolKeyPath+`\shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer cmdKey.Close()
	command, err := protocolCommand(exePath)
	if err != nil {
		return err
	}
	return cmdKey.SetStringValue("", command)
}

// ensureRegistration registers the shortcut and URL protocol on first use, and again
// whenever the binary has moved since the last registration.
func ensureRegistration(exePath, state string) error {
	marker := registrationMarkerPath(state)
	if data, err := os.ReadFile(marker); err == nil {
		var m registrationMarker
		if json.Unmarshal(data, &m) == nil && m.ExePath == exePath {
			return nil
		}
	}

	if err := ensureShortcut(exePath); err != nil {
		return err
	}
	if err := ensureProtocolRegistration(exePath); err != nil {
		return err
	}

	if err := os.MkdirAll(state, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(registrationMarker{ExePath: exePath})
	if err != nil {
		return err
	}
	return os.WriteFile(marker, data, 0o600)
}
