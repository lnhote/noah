// +build linux

package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"
)

type LockedFile struct{ *os.File }

func Exist(dirpath string) bool {
	names, err := ReadDir(dirpath)
	if err != nil {
		return false
	}
	return len(names) != 0
}

func ReadDir(dirpath string) ([]string, error) {
	dir, err := os.Open(dirpath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func TouchDirAll(dir string) error {
	// If path is already a directory, MkdirAll does nothing
	// and returns nil.
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		// if mkdirAll("a/text") and "text" is not
		// a directory, this will return syscall.ENOTDIR
		return err
	}
	return IsDirWriteable(dir)
}

func IsDirWriteable(dir string) error {
	f := filepath.Join(dir, ".touch")
	if err := WriteFile(f, []byte(""), 0600); err != nil {
		return err
	}
	return os.Remove(f)
}

func WriteFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

// CreateDirAll is similar to TouchDirAll but returns error
// if the deepest directory was not empty.
func CreateDirAll(dir string) error {
	err := TouchDirAll(dir)
	if err == nil {
		var ns []string
		ns, err = ReadDir(dir)
		if err != nil {
			return err
		}
		if len(ns) != 0 {
			err = fmt.Errorf("expected %q to be empty, got %q", dir, ns)
		}
	}
	return err
}

func Preallocate(f *os.File, sizeInBytes int64, extendFile bool) error {
	if sizeInBytes == 0 {
		// fallocate will return EINVAL if length is 0; skip
		return nil
	}
	if extendFile {
		return preallocExtend(f, sizeInBytes)
	}
	return preallocFixed(f, sizeInBytes)
}

func preallocExtend(f *os.File, sizeInBytes int64) error {
	if err := preallocFixed(f, sizeInBytes); err != nil {
		return err
	}
	return preallocExtendTrunc(f, sizeInBytes)
}

func preallocExtendTrunc(f *os.File, sizeInBytes int64) error {
	curOff, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	size, err := f.Seek(sizeInBytes, io.SeekEnd)
	if err != nil {
		return err
	}
	if _, err = f.Seek(curOff, io.SeekStart); err != nil {
		return err
	}
	if sizeInBytes > size {
		return nil
	}
	return f.Truncate(sizeInBytes)
}

func preallocFixed(f *os.File, sizeInBytes int64) error {
	// use mode = 1 to keep size; see FALLOC_FL_KEEP_SIZE
	err := syscall.Fallocate(int(f.Fd()), 1, 0, sizeInBytes)
	if err != nil {
		errno, ok := err.(syscall.Errno)
		// treat not supported as nil error
		if ok && errno == syscall.ENOTSUP {
			return nil
		}
	}
	return err
}

func LockFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	return flockLockFile(path, flag, perm)
}

func flockLockFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return &LockedFile{f}, err
}

func Fsync(f *os.File) error {
	return f.Sync()
}

func WriteAndSyncFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = Fsync(f)
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}
