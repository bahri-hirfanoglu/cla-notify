package present

import "testing"

func TestProtocolCommand(t *testing.T) {
	got, err := protocolCommand(`C:\Program Files\cla-notify\cla-notify.exe`)
	if err != nil {
		t.Fatalf("protocolCommand() error = %v", err)
	}
	want := `"C:\Program Files\cla-notify\cla-notify.exe" focus-url "%1"`
	if got != want {
		t.Fatalf("protocolCommand() = %q, want %q", got, want)
	}
}

func TestProtocolCommandRejectsDoubleQuote(t *testing.T) {
	if _, err := protocolCommand(`C:\Program Files\cla-notify\evil".exe`); err == nil {
		t.Fatal("protocolCommand() expected an error for an exePath containing a double quote")
	}
}
