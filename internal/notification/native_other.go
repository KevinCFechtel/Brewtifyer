//go:build !darwin

package notification

type NativeSender struct{}

func NewNativeSender() Sender {
	return NativeSender{}
}

func (NativeSender) Send(string, string) {}
