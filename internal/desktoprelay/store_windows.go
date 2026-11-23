package desktoprelay

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

type storedScope struct {
	Scope
	Secret string `json:"secret"`
}

type diskState struct {
	Version              int                            `json:"version"`
	Enabled              bool                           `json:"enabled"`
	Certificate          string                         `json:"certificate"`
	PrivateKey           string                         `json:"private_key"`
	Scopes               map[string]storedScope         `json:"scopes"`
	Sessions             map[string]Session             `json:"sessions"`
	ConversationBindings map[string]ConversationBinding `json:"conversation_bindings,omitempty"`
	Setups               map[string]setupRecord         `json:"setups,omitempty"`
}

type privateStore struct {
	dir           *os.File
	syncDirectory func(*os.File) error
}

var errCommitUncertain = errors.New("relay state commit durability uncertain")
var errWindowsRelay = errors.New("Claude Desktop routing is not available in the Windows preview")

func (s *privateStore) Close() error                { return s.dir.Close() }
func (s *privateStore) Remove(string) error         { return errWindowsRelay }
func (s *privateStore) Rename(string, string) error { return errWindowsRelay }
func (s *privateStore) OpenFile(string, int, os.FileMode) (*os.File, error) {
	return nil, errWindowsRelay
}
func openStore(string, bool) (*privateStore, error)     { return nil, errWindowsRelay }
func readPrivate(*privateStore, string) ([]byte, error) { return nil, errWindowsRelay }
func atomicPrivate(*privateStore, string, []byte) error { return errWindowsRelay }
func loadState(root *privateStore) (diskState, bool, error) {
	b, err := readPrivate(root, "state.json")
	if os.IsNotExist(err) {
		return diskState{}, false, nil
	}
	if err != nil {
		return diskState{}, false, err
	}
	var s diskState
	if err = json.Unmarshal(b, &s); err != nil || s.Version != 1 {
		return s, false, errors.New("unsupported relay state")
	}
	if !validSetupRecords(s, root.dir.Name()) {
		return s, false, errors.New("corrupt relay setup journal")
	}
	if s.Scopes == nil {
		s.Scopes = map[string]storedScope{}
	}
	if s.Sessions == nil {
		s.Sessions = map[string]Session{}
	}
	pruneSessions(&s, time.Now())
	// Drop records that no longer belong to a scope instead of refusing to start.
	for key, session := range s.Sessions {
		if _, ok := s.Scopes[session.ScopeID]; !ok || key != taskKey(session.ScopeID, session.SessionID) {
			delete(s.Sessions, key)
			continue
		}
		session.InFlight = 0
		s.Sessions[key] = session
	}
	for key, binding := range s.ConversationBindings {
		if _, ok := s.Scopes[binding.ScopeID]; !ok || key != taskKey(binding.ScopeID, binding.ConversationID) {
			delete(s.ConversationBindings, key)
		}
	}
	return s, true, nil
}

// Sessions are request telemetry. Idle ones are forgotten after two weeks and
// at most maxSessions are kept, so the state file stays small.
const (
	maxSessions = 2000
	sessionTTL  = 14 * 24 * time.Hour
)

func pruneSessions(s *diskState, now time.Time) {
	if len(s.Sessions) == 0 {
		return
	}
	keys := make([]string, 0, len(s.Sessions))
	for key, session := range s.Sessions {
		if session.InFlight == 0 && now.Sub(session.LastSeen) > sessionTTL {
			delete(s.Sessions, key)
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) <= maxSessions {
		return
	}
	sort.Slice(keys, func(i, j int) bool { return s.Sessions[keys[i]].LastSeen.After(s.Sessions[keys[j]].LastSeen) })
	for _, key := range keys[maxSessions:] {
		if s.Sessions[key].InFlight == 0 {
			delete(s.Sessions, key)
		}
	}
}

// saveLocked persists the state now. The caller holds m.mu.
func (m *Manager) saveLocked() error {
	if m.store == nil {
		return ErrUnavailable
	}
	b, seq, err := m.snapshotLocked()
	if err == nil {
		err = m.write(m.store, b, seq)
	}
	if err != nil {
		return fmt.Errorf("%w: could not persist state", ErrUnavailable)
	}
	return nil
}

func (m *Manager) snapshotLocked() ([]byte, uint64, error) {
	pruneSessions(&m.state, time.Now())
	for key := range m.threads {
		if _, ok := m.state.Sessions[key]; !ok {
			delete(m.threads, key)
		}
	}
	m.saveSeq++
	b, err := json.Marshal(m.state)
	return b, m.saveSeq, err
}

func (m *Manager) write(store *privateStore, b []byte, seq uint64) error {
	m.saveMu.Lock()
	defer m.saveMu.Unlock()
	if seq < m.savedSeq {
		return nil
	}
	if err := atomicPrivate(store, "state.json", b); err != nil {
		return err
	}
	m.savedSeq = seq
	return nil
}

// saveSoonLocked batches persistence of request observations. The write runs
// outside m.mu, so requests never wait for, or fail because of, the disk.
func (m *Manager) saveSoonLocked() {
	if m.savePending {
		return
	}
	m.savePending = true
	time.AfterFunc(2*time.Second, func() {
		m.mu.Lock()
		m.savePending = false
		store := m.store
		if store == nil {
			m.mu.Unlock()
			return
		}
		b, seq, err := m.snapshotLocked()
		m.mu.Unlock()
		if err == nil {
			err = m.write(store, b, seq)
		}
		if err != nil {
			log.Printf("desktop relay: save state: %v", err)
		}
	})
}

func rollbackAllowed(err error) bool { return !errors.Is(err, errCommitUncertain) }

func taskKey(scope, session string) string { return scope + "/" + session }

func validUUID(s string) bool {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	zero := true
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
		if c != '0' {
			zero = false
		}
	}
	return !zero
}
