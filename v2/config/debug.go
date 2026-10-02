package config

import (
	"fmt"
	"runtime/debug"
	"time"
)

func DeferPanicToError(name string, err func(error)) {
	if r := recover(); r != nil {
		s := fmt.Errorf("%s panic: %s\n%s", name, r, string(debug.Stack()))
		err(s)
		<-time.After(5 * time.Second)
	}
}
