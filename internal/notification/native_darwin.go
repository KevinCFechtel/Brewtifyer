//go:build darwin

package notification

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework UserNotifications
#include <stdlib.h>

void BrewtifyerSendNotification(const char *title, const char *body);
*/
import "C"

import "unsafe"

type NativeSender struct{}

func NewNativeSender() Sender {
	return NativeSender{}
}

func (NativeSender) Send(title, body string) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	cBody := C.CString(body)
	defer C.free(unsafe.Pointer(cBody))
	C.BrewtifyerSendNotification(cTitle, cBody)
}
