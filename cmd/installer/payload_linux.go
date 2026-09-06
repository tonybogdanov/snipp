//go:build linux

package main

import _ "embed"

//go:embed payload/linux.bin
var appBinary []byte
