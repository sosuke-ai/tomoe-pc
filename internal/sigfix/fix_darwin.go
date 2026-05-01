//go:build darwin

package sigfix

// AfterSherpa is a no-op on darwin in this Phase 0 stub.
//
// macOS uses POSIX signals like Linux, so the SA_ONSTACK fix may apply here
// too. Phase 2.2 will investigate ORT signal-handler behavior on darwin and
// either port fix_linux.go's C function or leave this as a no-op if not
// needed. For now the Phase 0 build target is GOOS=darwin CGO_ENABLED=0,
// so cgo isn't available anyway.
func AfterSherpa() {}
