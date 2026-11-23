// Package winfs provides handle-based private file access on Windows.
package winfs

import (
	"crypto/rand"
	"errors"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

func userSID() (*windows.SID, error) {
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	return u.User.Sid, nil
}

// Check requires an owner-controlled object and a DACL granting access only
// to its owner, SYSTEM and Administrators. Unknown ACE formats fail closed.
func Check(f *os.File) error {
	sd, err := windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return err
	}
	me, err := userSID()
	if err != nil {
		return err
	}
	if owner == nil || owner.String() != me.String() {
		return errors.New("file is not owned by the current Windows user")
	}
	acl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	if acl == nil {
		return errors.New("file has no protective Windows DACL")
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceFlags&0x08 != 0 {
			continue
		} // inherit-only entries do not grant access here
		if ace.Header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
			continue
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return errors.New("unsupported Windows ACL entry")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if ace.Mask != 0 && sid.String() != me.String() && !sid.IsWellKnown(windows.WinLocalSystemSid) && !sid.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
			return errors.New("file permissions allow access by another Windows principal")
		}
	}
	return nil
}

// Protect is used only for Switcher's own data directory and newly created
// files, never to silently rewrite a native CLI's existing permissions.
func Protect(f *os.File) error {
	me, err := userSID()
	if err != nil {
		return err
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + me.String() + ")(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if err != nil {
		return err
	}
	acl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}

// OpenRoot pins a local directory with Go's Windows os.Root implementation.
// The second open rejects reparse points; file identities must match before use.
func OpenRoot(path string, create, protect bool) (*os.Root, error) {
	if !filepath.IsAbs(path) || strings.HasPrefix(path, `\\`) {
		return nil, errors.New("a local absolute Windows path is required")
	}
	if create {
		if err := os.MkdirAll(path, 0700); err != nil {
			return nil, err
		}
	}
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	access := uint32(windows.READ_CONTROL | windows.FILE_READ_ATTRIBUTES)
	if protect {
		access |= windows.WRITE_DAC
	}
	h, err := windows.CreateFile(ptr, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	pin := os.NewFile(uintptr(h), path)
	defer pin.Close()
	var attrs windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &attrs); err != nil {
		return nil, err
	}
	if attrs.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || attrs.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		return nil, errors.New("data directory is redirected or not a directory")
	}
	if protect { // Verify ownership before changing this object's DACL.
		sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
		if err != nil {
			return nil, err
		}
		owner, _, err := sd.Owner()
		if err != nil {
			return nil, err
		}
		me, err := userSID()
		if err != nil {
			return nil, err
		}
		if owner == nil || owner.String() != me.String() {
			return nil, errors.New("data directory is not owned by the current user")
		}
		if err := Protect(pin); err != nil {
			return nil, err
		}
	}
	if err := Check(pin); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	f, err := root.Open(".")
	if err != nil {
		root.Close()
		return nil, err
	}
	defer f.Close()
	a, e1 := pin.Stat()
	b, e2 := f.Stat()
	if e1 != nil || e2 != nil || !os.SameFile(a, b) {
		root.Close()
		return nil, errors.New("directory changed while opening")
	}
	return root, nil
}

func Open(root *os.Root, name string, flags int, mode os.FileMode) (*os.File, error) {
	if filepath.Base(name) != name || name == "." || strings.ContainsAny(name, `/\:`) {
		return nil, errors.New("invalid private file name")
	}
	info, err := root.Lstat(name)
	if err == nil && !info.Mode().IsRegular() {
		return nil, errors.New("private file is redirected or not regular")
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	var f *os.File
	if flags&os.O_CREATE != 0 && flags&os.O_EXCL != 0 {
		// Create with a restrictive descriptor in the kernel, before any bytes
		// can be written. The parent comes from an already pinned os.Root.
		parent, e := root.Open(".")
		if e != nil {
			return nil, e
		}
		defer parent.Close()
		me, e := userSID()
		if e != nil {
			return nil, e
		}
		sd, e := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + me.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)")
		if e != nil {
			return nil, e
		}
		objectName, e := windows.NewNTUnicodeString(name)
		if e != nil {
			return nil, e
		}
		oa := &windows.OBJECT_ATTRIBUTES{RootDirectory: windows.Handle(parent.Fd()), ObjectName: objectName, Attributes: windows.OBJ_CASE_INSENSITIVE, SecurityDescriptor: sd}
		oa.Length = uint32(unsafe.Sizeof(*oa))
		var h windows.Handle
		var iosb windows.IO_STATUS_BLOCK
		access := uint32(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE | windows.WRITE_DAC)
		e = windows.NtCreateFile(&h, access, oa, &iosb, nil, windows.FILE_ATTRIBUTE_NORMAL, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, windows.FILE_CREATE, windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
		if status, ok := e.(windows.NTStatus); ok {
			e = status.Errno()
		}
		if e != nil {
			return nil, &os.PathError{Op: "create", Path: name, Err: e}
		}
		f = os.NewFile(uintptr(h), name)
	} else {
		f, err = root.OpenFile(name, flags, mode)
		if err != nil {
			return nil, err
		}
	}
	fail := func(err error) (*os.File, error) { f.Close(); return nil, err }
	opened, err := f.Stat()
	if err != nil {
		return fail(err)
	}
	if !opened.Mode().IsRegular() || info != nil && !os.SameFile(info, opened) {
		return fail(errors.New("private file changed while opening"))
	}
	var stat windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &stat); err != nil {
		return fail(err)
	}
	if stat.NumberOfLinks != 1 {
		return fail(errors.New("hard-linked private files are refused"))
	}

	if err := Check(f); err != nil {
		return fail(err)
	}
	return f, nil
}

func Read(root *os.Root, name string, limit int64) ([]byte, error) {
	f, err := Open(root, name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(raw)) > limit {
		return nil, errors.New("private file exceeds its size limit")
	}
	return raw, err
}

func Temp(root *os.Root, data []byte) (string, error) {
	name := ".switcher-" + rand.Text() + ".tmp"
	f, err := Open(root, name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		root.Remove(name)
		return "", err
	}
	return name, nil
}

func Atomic(root *os.Root, name string, data []byte) error {
	if f, err := Open(root, name, os.O_RDONLY, 0); err == nil {
		f.Close()
	} else if !os.IsNotExist(err) {
		return err
	}
	tmp, err := Temp(root, data)
	if err != nil {
		return err
	}
	defer root.Remove(tmp)
	return root.Rename(tmp, name)
}
