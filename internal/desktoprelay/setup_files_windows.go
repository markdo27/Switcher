package desktoprelay

import (
	"context"
	"errors"
	"os"
)

var errSetupMissing = errors.New("setup path missing")

type setupFile struct {
	data   []byte
	exists bool
	info   os.FileInfo
}
type setupLease struct{}

func openSetupDirectory(string, bool) (*os.File, error) { return nil, errWindowsRelay }
func setupDirectoryCurrent(*os.File) bool               { return false }
func readSetupFile(*os.File, string) (setupFile, error) { return setupFile{}, errWindowsRelay }
func setupSameFile(setupFile, setupFile) bool           { return false }
func acquireSetupLease(context.Context, *os.File, string) (*setupLease, error) {
	return nil, errWindowsRelay
}
func (l *setupLease) check() bool { return false }
func (l *setupLease) close()      {}
func (m *Manager) writeSetupFile(context.Context, *os.File, string, *setupLease, setupFile, []byte) error {
	return errWindowsRelay
}
func (m *Manager) setupCommitUncertainLocked() error                { return errWindowsRelay }
func (m *Manager) setupBackupDirectory(bool) (*privateStore, error) { return nil, errWindowsRelay }
func (m *Manager) backupSetupLocked(r setupRecord, data []byte) error {
	root, err := m.setupBackupDirectory(true)
	if err != nil {
		return err
	}
	defer root.Close()
	return atomicPrivate(root, r.BackupName, data)
}
func (m *Manager) validateSetupBackupLocked(r setupRecord) error {
	_, err := m.loadSetupBaselineLocked(r)
	return err
}

func (m *Manager) loadSetupBaselineLocked(r setupRecord) (setupDocument, error) {
	root, err := m.setupBackupDirectory(false)
	if err != nil {
		return setupDocument{}, setupFailure(err)
	}
	defer root.Close()
	b, err := readPrivate(root, r.BackupName)
	if err != nil || setupDigest(b, r.FileExisted) != r.OriginalDigest {
		return setupDocument{}, setupError("setup_unavailable", ErrUnavailable)
	}
	doc, err := parseSetupDocument(b, r.FileExisted)
	if err != nil || doc.shape != r.EnvShape || !doc.matches(r.OriginalEnv) {
		return setupDocument{}, setupError("setup_unavailable", ErrUnavailable)
	}
	return doc, nil
}
func (m *Manager) readSetupState() (diskState, error) {
	dir, err := openSetupDirectory(m.cfg.DataRoot, false)
	if errors.Is(err, errSetupMissing) {
		return diskState{}, nil
	}
	if err != nil {
		return diskState{}, err
	}
	defer dir.Close()
	info, err := dir.Stat()
	if err != nil || info.Mode().Perm() != 0700 {
		return diskState{}, ErrUnavailable
	}
	s, _, err := loadState(&privateStore{dir: dir})
	return s, err
}
