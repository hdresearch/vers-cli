package ssh

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-vm-id", "/path/to/key")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.host != "test-vm-id" {
		t.Errorf("expected host 'test-vm-id', got %q", client.host)
	}
	if client.keyPath != "/path/to/key" {
		t.Errorf("expected keyPath '/path/to/key', got %q", client.keyPath)
	}
	if client.vmDomain != DefaultVMDomain {
		t.Errorf("expected vmDomain %q, got %q", DefaultVMDomain, client.vmDomain)
	}
}

func TestNewClient_CustomDomain(t *testing.T) {
	client := NewClient("test-vm-id", "/path/to/key", "vm.staging.vers.sh")
	if client.vmDomain != "vm.staging.vers.sh" {
		t.Errorf("expected vmDomain 'vm.staging.vers.sh', got %q", client.vmDomain)
	}
}

func TestClient_Hostname(t *testing.T) {
	client := NewClient("abc123", "/key")
	hostname := client.hostname()
	expected := "abc123.vm.vers.sh"
	if hostname != expected {
		t.Errorf("expected hostname %q, got %q", expected, hostname)
	}
}

func TestClient_Hostname_Staging(t *testing.T) {
	client := NewClient("abc123", "/key", "vm.staging.vers.sh")
	hostname := client.hostname()
	expected := "abc123.vm.staging.vers.sh"
	if hostname != expected {
		t.Errorf("expected hostname %q, got %q", expected, hostname)
	}
}

func TestPortToString(t *testing.T) {
	tests := []struct {
		port     int
		expected string
	}{
		{22, "22"},
		{443, "443"},
		{2222, "2222"},
	}
	for _, tc := range tests {
		got := PortToString(tc.port)
		if got != tc.expected {
			t.Errorf("PortToString(%d) = %q, want %q", tc.port, got, tc.expected)
		}
	}
}
