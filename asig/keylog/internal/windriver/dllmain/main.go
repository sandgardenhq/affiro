//go:build windows

package main

//#include "dllmain.h"
import "C"

//export Install
func Install() {
	C.install()
}

//export Uninstall
func Uninstall() {
	C.uninstall()
}

func main() {
	// golang based DLLs need to be a main package,
	// and so need a main function, but we can't access the variable we need to register hooks
	// without C's DllMain signatures. So, this thin Go wrapper offers little
	// over the C its wrapping, other than allowing us to build the C with a Go toolchain.
	// A future version of this DLL could try to use syscalls to get the DLL's HINSTANCE
	// within Go, and then move all of the logic to Go from C, or remove this wrapper and
	// build the DLL directly with gcc, etc.
}
