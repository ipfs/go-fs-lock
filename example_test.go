package fslock_test

import (
	"errors"
	"fmt"

	"github.com/ipfs/go-fs-lock"
)

func ExampleLockedError() {
	_, err := fslock.Lock("/tmp/", "foo.lock")
	fmt.Println("locked:", errors.As(err, new(fslock.LockedError)))

	_, err = fslock.Lock("/tmp/", "foo.lock")
	fmt.Println("locked:", errors.As(err, new(fslock.LockedError)))
	// Output:
	// locked: false
	// locked: true
}
