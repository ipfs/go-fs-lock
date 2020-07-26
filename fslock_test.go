package fslock_test

import (
	"bufio"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	lock "github.com/ipfs/go-fs-lock"
)

func assertLock(t *testing.T, confdir, lockFile string, expected bool) {
	t.Helper()

	isLocked, err := lock.Locked(confdir, lockFile)
	if err != nil {
		t.Fatal(err)
	}

	if isLocked != expected {
		t.Fatalf("expected %t to be %t", isLocked, expected)
	}
}

func TestLockSimple(t *testing.T) {
	lockFile := "my-test.lock"
	confdir := os.TempDir()

	// make sure we start clean
	_ = os.Remove(path.Join(confdir, lockFile))

	assertLock(t, confdir, lockFile, false)

	lockfile, err := lock.Lock(confdir, lockFile)
	if err != nil {
		t.Fatal(err)
	}

	assertLock(t, confdir, lockFile, true)

	if err := lockfile.Close(); err != nil {
		t.Fatal(err)
	}

	assertLock(t, confdir, lockFile, false)

	// second round of locking

	lockfile, err = lock.Lock(confdir, lockFile)
	if err != nil {
		t.Fatal(err)
	}

	assertLock(t, confdir, lockFile, true)

	if err := lockfile.Close(); err != nil {
		t.Fatal(err)
	}

	assertLock(t, confdir, lockFile, false)
}

func TestLockMultiple(t *testing.T) {
	lockFile1 := "test-1.lock"
	lockFile2 := "test-2.lock"
	confdir := os.TempDir()

	// make sure we start clean
	_ = os.Remove(path.Join(confdir, lockFile1))
	_ = os.Remove(path.Join(confdir, lockFile2))

	lock1, err := lock.Lock(confdir, lockFile1)
	if err != nil {
		t.Fatal(err)
	}
	lock2, err := lock.Lock(confdir, lockFile2)
	if err != nil {
		t.Fatal(err)
	}

	assertLock(t, confdir, lockFile1, true)
	assertLock(t, confdir, lockFile2, true)

	if err := lock1.Close(); err != nil {
		t.Fatal(err)
	}

	assertLock(t, confdir, lockFile1, false)
	assertLock(t, confdir, lockFile2, true)

	if err := lock2.Close(); err != nil {
		t.Fatal(err)
	}

	assertLock(t, confdir, lockFile1, false)
	assertLock(t, confdir, lockFile2, false)
}

func TestLockedByOthers(t *testing.T) {
	const (
		lockedMsg = "locked\n"
		lockFile  = "my-test.lock"
		wantErr   = "someone else has the lock"
	)

	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" { // child process
		confdir := os.Args[3]
		if _, err := lock.Lock(confdir, lockFile); err != nil {
			t.Fatalf("child lock: %v", err)
		}
		os.Stdout.WriteString(lockedMsg)
		time.Sleep(10 * time.Minute)
		return
	}

	confdir, err := ioutil.TempDir("", "go-fs-lock-test")
	if err != nil {
		t.Fatalf("creating temporary directory: %v", err)
	}
	defer os.RemoveAll(confdir)

	// Execute a child process that locks the file.
	cmd := exec.Command(os.Args[0], "-test.run=TestLockedByOthers", "--", confdir)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("cmd.StdoutPipe: %v", err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}
	defer cmd.Process.Kill()

	// Wait for the child to lock the file.
	b := bufio.NewReader(stdout)
	line, err := b.ReadString('\n')
	if err != nil {
		t.Fatalf("read from child: %v", err)
	}
	if got, want := line, lockedMsg; got != want {
		t.Fatalf("got %q from child; want %q", got, want)
	}

	// Parent should not be able to lock the file.
	_, err = lock.Lock(confdir, lockFile)
	if err == nil {
		t.Fatalf("parent successfully acquired the lock")
	}
	pe, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("wrong error type %T", err)
	}
	if got := pe.Error(); !strings.Contains(got, wantErr) {
		t.Fatalf("error %q does not contain %q", got, wantErr)
	}
}
