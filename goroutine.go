package services

import (
	"runtime"
	"strconv"
	"strings"
)

// goid returns the current goroutine ID.
// This is used for tracking resolution chains in concurrent operations.
func goid() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, _ := strconv.ParseInt(idField, 10, 64)
	return id
}
