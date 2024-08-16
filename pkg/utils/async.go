package utils

import "fmt"

func GoSafe(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("go routine recovered from panic: %s\n", r)
			}
		}()

		fn()
	}()
}
