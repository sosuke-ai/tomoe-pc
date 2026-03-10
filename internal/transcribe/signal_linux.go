package transcribe

/*
#include <signal.h>

// fixOnnxSignalHandler re-installs the SIGSEGV handler with SA_ONSTACK.
// ONNX Runtime installs a SIGSEGV handler without SA_ONSTACK, which
// conflicts with Go's signal handling and causes "non-Go code set up
// signal handler without SA_ONSTACK flag" fatal errors.
static void fixOnnxSignalHandler() {
	struct sigaction sa;
	if (sigaction(SIGSEGV, NULL, &sa) == 0) {
		if (sa.sa_handler != SIG_DFL && sa.sa_handler != SIG_IGN) {
			sa.sa_flags |= SA_ONSTACK;
			sigaction(SIGSEGV, &sa, NULL);
		}
	}
}
*/
import "C"

// fixSignalHandlers patches ONNX Runtime's signal handlers to be
// compatible with Go's runtime. Must be called after sherpa-onnx init.
func fixSignalHandlers() {
	C.fixOnnxSignalHandler()
}
