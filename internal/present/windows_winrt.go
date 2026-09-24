//go:build windows

package present

import (
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
)

// GUIDs for the WinRT interfaces cla-notify calls into, sourced from the published
// Windows Runtime IDL (windows.ui.notifications.idl, windows.data.xml.dom.idl,
// windows.foundation.idl). iidIReferenceDateTime is the parameterized IReference<DateTime>
// interface id, computed with the documented WinRT pinterface hashing algorithm and
// cross-checked against real published IReference<BYTE>/IReference<boolean> IIDs.
var (
	iidIXmlDocument                     = ole.NewGUID("f7f3a506-1e87-42d6-bcfb-b8c809fa5494")
	iidIXmlDocumentIO                   = ole.NewGUID("6cd0e74e-ee65-4489-9ebf-ca43e87ba637")
	iidIToastNotificationManagerStatics = ole.NewGUID("50ac103f-d235-4598-bbef-98fe4d1a3ad4")
	iidIToastNotificationFactory        = ole.NewGUID("04124b20-82c6-4229-b109-fd9ed4662b53")
	iidIToastNotifier                   = ole.NewGUID("75927b93-03f3-41ec-91d3-6e5bac1b38e7")
	iidIToastNotification2              = ole.NewGUID("9dfb9fd1-143a-490e-90bf-b9fba7132de7")
	iidIReferenceDateTime               = ole.NewGUID("5541d8a7-497c-5aa4-86fc-7713adbf2a2c")
	iidIPropertyValueStatics            = ole.NewGUID("629bdbc8-d932-4ff4-96b9-8d96c5c1e858")
)

const (
	rtXmlDocument              = "Windows.Data.Xml.Dom.XmlDocument"
	rtToastNotificationManager = "Windows.UI.Notifications.ToastNotificationManager"
	rtToastNotification        = "Windows.UI.Notifications.ToastNotification"
	rtPropertyValue            = "Windows.Foundation.PropertyValue"
)

var (
	winrtOnce sync.Once
	winrtErr  error
)

// ensureWinRT initializes COM and the Windows Runtime once per process.
func ensureWinRT() error {
	winrtOnce.Do(func() {
		// RoInitialize also initializes COM; go-ole reports S_FALSE ("already initialized") as an error.
		winrtErr = ignoreAlreadyInitialized(ole.RoInitialize(1)) // RO_INIT_MULTITHREADED
	})
	return winrtErr
}

// callMethod invokes a raw COM vtable slot and turns a failing HRESULT into an error.
func callMethod(method uintptr, args ...uintptr) error {
	r1, _, _ := syscall.SyscallN(method, args...)
	if int32(r1) != 0 {
		return ole.NewError(r1)
	}
	return nil
}

// --- IXmlDocument (marker type passed into IToastNotificationFactory.CreateToastNotification) ---

type xmlDocument struct{ ole.IInspectable }

// --- IXmlDocumentIO ---

type xmlDocumentIOVtbl struct {
	ole.IInspectableVtbl
	LoadXml             uintptr
	LoadXmlWithSettings uintptr
	SaveToFileAsync     uintptr
}

type xmlDocumentIO struct{ ole.IInspectable }

func (v *xmlDocumentIO) vtbl() *xmlDocumentIOVtbl {
	return (*xmlDocumentIOVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *xmlDocumentIO) LoadXml(xml ole.HString) error {
	return callMethod(v.vtbl().LoadXml, uintptr(unsafe.Pointer(v)), uintptr(xml))
}

// --- IToastNotificationManagerStatics ---

type toastNotificationManagerStaticsVtbl struct {
	ole.IInspectableVtbl
	CreateToastNotifier       uintptr
	CreateToastNotifierWithId uintptr
	GetTemplateContent        uintptr
}

type toastNotificationManagerStatics struct{ ole.IInspectable }

func (v *toastNotificationManagerStatics) vtbl() *toastNotificationManagerStaticsVtbl {
	return (*toastNotificationManagerStaticsVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *toastNotificationManagerStatics) CreateToastNotifierWithId(appID ole.HString) (*toastNotifier, error) {
	var out *toastNotifier
	err := callMethod(v.vtbl().CreateToastNotifierWithId, uintptr(unsafe.Pointer(v)), uintptr(appID), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// --- IToastNotificationFactory ---

type toastNotificationFactoryVtbl struct {
	ole.IInspectableVtbl
	CreateToastNotification uintptr
}

type toastNotificationFactory struct{ ole.IInspectable }

func (v *toastNotificationFactory) vtbl() *toastNotificationFactoryVtbl {
	return (*toastNotificationFactoryVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *toastNotificationFactory) CreateToastNotification(doc *xmlDocument) (*toastNotification, error) {
	var out *toastNotification
	err := callMethod(v.vtbl().CreateToastNotification, uintptr(unsafe.Pointer(v)), uintptr(unsafe.Pointer(doc)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// --- IToastNotification ---

type toastNotificationVtbl struct {
	ole.IInspectableVtbl
	GetContent        uintptr
	PutExpirationTime uintptr
	GetExpirationTime uintptr
}

type toastNotification struct{ ole.IInspectable }

func (v *toastNotification) vtbl() *toastNotificationVtbl {
	return (*toastNotificationVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *toastNotification) PutExpirationTime(ref *ole.IInspectable) error {
	return callMethod(v.vtbl().PutExpirationTime, uintptr(unsafe.Pointer(v)), uintptr(unsafe.Pointer(ref)))
}

func (v *toastNotification) queryNotification2() (*toastNotification2, error) {
	var out *toastNotification2
	err := v.IUnknown.PutQueryInterface(iidIToastNotification2, &out)
	return out, err
}

// --- IToastNotification2 (Tag/Group) ---

type toastNotification2Vtbl struct {
	ole.IInspectableVtbl
	PutTag   uintptr
	GetTag   uintptr
	PutGroup uintptr
	GetGroup uintptr
}

type toastNotification2 struct{ ole.IInspectable }

func (v *toastNotification2) vtbl() *toastNotification2Vtbl {
	return (*toastNotification2Vtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *toastNotification2) PutTag(s ole.HString) error {
	return callMethod(v.vtbl().PutTag, uintptr(unsafe.Pointer(v)), uintptr(s))
}

func (v *toastNotification2) PutGroup(s ole.HString) error {
	return callMethod(v.vtbl().PutGroup, uintptr(unsafe.Pointer(v)), uintptr(s))
}

// --- IToastNotifier ---

type toastNotifierVtbl struct {
	ole.IInspectableVtbl
	Show uintptr
}

type toastNotifier struct{ ole.IInspectable }

func (v *toastNotifier) vtbl() *toastNotifierVtbl {
	return (*toastNotifierVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *toastNotifier) Show(n *toastNotification) error {
	return callMethod(v.vtbl().Show, uintptr(unsafe.Pointer(v)), uintptr(unsafe.Pointer(n)))
}

// --- IPropertyValueStatics (only through CreateDateTime, used to box ExpirationTime) ---

type propertyValueStaticsVtbl struct {
	ole.IInspectableVtbl
	CreateEmpty       uintptr
	CreateUInt8       uintptr
	CreateInt16       uintptr
	CreateUInt16      uintptr
	CreateInt32       uintptr
	CreateUInt32      uintptr
	CreateInt64       uintptr
	CreateUInt64      uintptr
	CreateSingle      uintptr
	CreateDouble      uintptr
	CreateChar16      uintptr
	CreateBoolean     uintptr
	CreateString      uintptr
	CreateInspectable uintptr
	CreateGuid        uintptr
	CreateDateTime    uintptr
}

type propertyValueStatics struct{ ole.IInspectable }

func (v *propertyValueStatics) vtbl() *propertyValueStaticsVtbl {
	return (*propertyValueStaticsVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *propertyValueStatics) CreateDateTime(universalTime int64) (*ole.IInspectable, error) {
	var out *ole.IInspectable
	err := callMethod(v.vtbl().CreateDateTime, uintptr(unsafe.Pointer(v)), uintptr(universalTime), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// setExpirationTime boxes when as a WinRT DateTime and sets it on n; failure is left to the
// caller to treat as best effort, since the toast is still useful without an expiration.
func setExpirationTime(n *toastNotification, when time.Time) error {
	propInsp, err := ole.RoGetActivationFactory(rtPropertyValue, iidIPropertyValueStatics)
	if err != nil {
		return err
	}
	defer propInsp.Release()
	propStatics := (*propertyValueStatics)(unsafe.Pointer(propInsp))

	boxed, err := propStatics.CreateDateTime(toWinRTTicks(when))
	if err != nil {
		return err
	}
	defer boxed.Release()

	var ref *ole.IInspectable
	if err := boxed.IUnknown.PutQueryInterface(iidIReferenceDateTime, &ref); err != nil {
		return err
	}
	defer ref.Release()

	return n.PutExpirationTime(ref)
}

// showToast renders xmlDoc, tags and groups it, and shows it through the WinRT toast notifier
// registered for appID.
func showToast(xmlDoc, appID, tag, group string, expires time.Time) error {
	if err := ensureWinRT(); err != nil {
		return err
	}

	xmlHString, err := ole.NewHString(xmlDoc)
	if err != nil {
		return err
	}
	defer ole.DeleteHString(xmlHString)

	docInsp, err := ole.RoActivateInstance(rtXmlDocument)
	if err != nil {
		return err
	}
	defer docInsp.Release()

	var doc *xmlDocument
	if err := docInsp.IUnknown.PutQueryInterface(iidIXmlDocument, &doc); err != nil {
		return err
	}
	defer doc.Release()

	var docIO *xmlDocumentIO
	if err := docInsp.IUnknown.PutQueryInterface(iidIXmlDocumentIO, &docIO); err != nil {
		return err
	}
	defer docIO.Release()

	if err := docIO.LoadXml(xmlHString); err != nil {
		return err
	}

	managerInsp, err := ole.RoGetActivationFactory(rtToastNotificationManager, iidIToastNotificationManagerStatics)
	if err != nil {
		return err
	}
	defer managerInsp.Release()
	manager := (*toastNotificationManagerStatics)(unsafe.Pointer(managerInsp))

	appIDHString, err := ole.NewHString(appID)
	if err != nil {
		return err
	}
	defer ole.DeleteHString(appIDHString)

	notifier, err := manager.CreateToastNotifierWithId(appIDHString)
	if err != nil {
		return err
	}
	defer notifier.Release()

	factoryInsp, err := ole.RoGetActivationFactory(rtToastNotification, iidIToastNotificationFactory)
	if err != nil {
		return err
	}
	defer factoryInsp.Release()
	factory := (*toastNotificationFactory)(unsafe.Pointer(factoryInsp))

	notification, err := factory.CreateToastNotification(doc)
	if err != nil {
		return err
	}
	defer notification.Release()

	if n2, err := notification.queryNotification2(); err == nil {
		if tagHString, err := ole.NewHString(tag); err == nil {
			_ = n2.PutTag(tagHString)
			ole.DeleteHString(tagHString)
		}
		if groupHString, err := ole.NewHString(group); err == nil {
			_ = n2.PutGroup(groupHString)
			ole.DeleteHString(groupHString)
		}
		n2.Release()
	}

	// Best effort: the toast is still shown without an expiration if this fails.
	_ = setExpirationTime(notification, expires)

	return notifier.Show(notification)
}
