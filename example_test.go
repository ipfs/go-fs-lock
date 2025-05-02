package fslock_test

import (
	"errors"
	"fmt"
	"os"

	fslock "github.com/ipfs/go-fs-lock"
)

func ExampleLockedError() {
	lockFile, err := fslock.Lock(os.TempDir(), "foo.lock")
	fmt.Println("locked:", errors.As(err, new(fslock.LockedError)))
	defer lockFile.Close()

	_, err = fslock.Lock(os.TempDir(), "foo.lock")
	fmt.Println("locked:", errors.As(err, new(fslock.LockedError)))
	// Output:
	// locked: false
	// locked: true
}
