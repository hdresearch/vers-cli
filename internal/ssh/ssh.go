// Package ssh provides native SSH connectivity over TLS for vers VMs.
//
// This package implements SSH-over-TLS tunneling where SSH connections
// are established through a TLS connection on port 443. VMs are accessed
// via DNS hostnames in the format {vm-id}.vm.vers.sh.
package ssh

import "strconv"

// PortToString converts a port number to string for compatibility.
func PortToString(port int) string { return strconv.Itoa(port) }
