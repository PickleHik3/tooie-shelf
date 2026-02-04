package graphics

import (
	"fmt"
	"os"
	"strings"
)

// debugEnabled enables verbose logging for icon extraction
var debugEnabled = os.Getenv("TOOIE_DEBUG") == "1"

// logDebug prints debug messages if debug mode is enabled
func logDebug(format string, args ...interface{}) {
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// logIconExtraction logs the full extraction flow for an app
func logIconExtraction(pkg string, step string, details ...string) {
	if !debugEnabled {
		return
	}
	detailStr := ""
	if len(details) > 0 {
		detailStr = " - " + strings.Join(details, ", ")
	}
	fmt.Fprintf(os.Stderr, "[ICON:%s] %s%s\n", pkg, step, detailStr)
}
