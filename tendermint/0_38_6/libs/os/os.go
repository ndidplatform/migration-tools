package os

import (
	"fmt"
	"os"
)

func Exit(s string) {
	fmt.Printf(s + "\n")
	os.Exit(1)
}
