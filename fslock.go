package fslock

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	util "github.com/ipfs/go-ipfs-util"
	logging "github.com/ipfs/go-log"
	lock "go4.org/lock"
)

// log is the fsrepo logger
var log = logging.Logger("lock")

func errPerm(path string) error {
	return fmt.Errorf("failed to take lock at %s: permission denied", path)
}

// Lock creates the lock.
func Lock(confdir, lockFile string) (io.Closer, error) {
	lk, err := lock.Lock(filepath.Join(confdir, lockFile))
	if err != nil {
		// EAGAIN == someone else has the lock
		if err == syscall.EAGAIN {
			return lk, errors.New(fmt.Sprintf("Someone else has the lock: %s", filepath.Join(confdir, lockFile)))
		}
		if strings.Contains(err.Error(), "resource temporarily unavailable") {
			return lk, errors.New(fmt.Sprintf("Someone else has the lock: %s", filepath.Join(confdir, lockFile)))
		}

		// we hold the lock ourselves
		if strings.Contains(err.Error(), "already locked") {
			return lk, errors.New(fmt.Sprintf("Lock is already held by us: %s", filepath.Join(confdir, lockFile)))
		}

		// lock fails on permissions error
		if os.IsPermission(err) {
			return lk, errPerm(confdir)
		}
		if isLockCreatePermFail(err) {
			return lk, errPerm(confdir)
		}
	}
	return lk, err
}

// Locked checks if there is a lock already set.
func Locked(confdir, lockFile string) (bool, error) {
	log.Debugf("Checking lock")
	if !util.FileExists(filepath.Join(confdir, lockFile)) {
		log.Debugf("File doesn't exist: %s", filepath.Join(confdir, lockFile))
		return false, nil
	}

	lk, err := Lock(confdir, lockFile)
	if err != nil {
		errCase := err.Error()
		if strings.Contains(errCase, "Lock is already held by us:") {
			log.Debugf(errCase)
			return true, nil
		}
		if strings.Contains(errCase, "Someone else has the lock:") {
			log.Debugf(errCase)
			return true, nil
		}

		// lock fails on permissions error
		if strings.Contains(errCase, "permissions denied") {
			log.Debugf(errCase)
			return false, err
		}

		// otherwise, we cant guarantee anything, error out
		return false, err
	}

	log.Debugf("No one has a lock")
	lk.Close()
	return false, nil
}

func isLockCreatePermFail(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Lock Create of") && strings.Contains(s, "permission denied")
}
