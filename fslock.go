package fslock

import (
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

type LockError struct {
	path string
	msg  string
}

type PermError LockError

func (e LockError) Error() string {
	return fmt.Sprintf("lock %s: %s", e.path, e.msg)
}

func (e PermError) Error() string {
	return LockError(e).Error()
}

// Lock creates the lock.
func Lock(confdir, lockFileName string) (io.Closer, error) {
	lockFilePath := filepath.Join(confdir, lockFileName)
	lk, err := lock.Lock(lockFilePath)
	if err != nil {
		// EAGAIN == someone else has the lock
		if err == syscall.EAGAIN {
			return lk, LockError{lockFilePath, "Someone else has the lock"}
		}
		if strings.Contains(err.Error(), "resource temporarily unavailable") {
			return lk, LockError{lockFilePath, "Someone else has the lock"}
		}

		// we hold the lock ourselves
		if strings.Contains(err.Error(), "already locked") {
			return lk, LockError{lockFilePath, "Lock is already held by us"}
		}

		// lock fails on permissions error
		if os.IsPermission(err) || isLockCreatePermFail(err) {
			return lk, PermError{confdir, "Permission denied"}
		}
	}
	return lk, err
}

// Locked checks if there is a lock already set.
func IsLocked(confdir, lockFile string) (bool, error) {
	log.Debugf("Checking lock")
	if !util.FileExists(filepath.Join(confdir, lockFile)) {
		log.Debugf("File doesn't exist: %s", filepath.Join(confdir, lockFile))
		return false, nil
	}

	lk, err := Lock(confdir, lockFile)
	if err == nil {
		log.Debugf("No one has a lock")
		lk.Close()
		return false, nil
	}

	switch err.(type) {
	case LockError:
		log.Debug(err)
		return true, nil
	case PermError:
		log.Debug(err)
		return false, err
	default:
		return false, err
	}
}

func isLockCreatePermFail(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Lock Create of") && strings.Contains(s, "permission denied")
}
