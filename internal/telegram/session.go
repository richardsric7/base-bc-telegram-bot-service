package telegram

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// flowKind identifies which multi-step conversation a chat is in the middle
// of.
type flowKind string

const (
	flowNone         flowKind = ""
	flowWalletImport flowKind = "wallet_import"
	flowTokenCreate  flowKind = "token_create"
)

// session holds multi-step conversation state for a single chat.
type session struct {
	Flow flowKind
	Step string
	Data map[string]string
}

// sessionStore is a simple in-memory, mutex-guarded map of chat ID ->
// session. State is not persisted across restarts, which is acceptable for
// short guided flows (the operator just re-runs the command).
type sessionStore struct {
	mu       sync.Mutex
	sessions map[int64]*session
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[int64]*session)}
}

func (s *sessionStore) start(chatID int64, flow flowKind, step string) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := &session{Flow: flow, Step: step, Data: make(map[string]string)}
	s.sessions[chatID] = sess
	return sess
}

func (s *sessionStore) get(chatID int64) (*session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[chatID]
	return sess, ok
}

func (s *sessionStore) clear(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, chatID)
}

// pendingAction is a confirmable, state-changing operation queued from a
// command handler and executed only if the user taps "Confirm" on the
// inline keyboard reply.
type pendingAction struct {
	UserTelegramID int64
	ChatID         int64
	Description    string
	Execute        func() (string, error)
	CreatedAt      time.Time
}

// pendingStore tracks pending confirmations by a random token embedded in
// the inline keyboard's callback data.
type pendingStore struct {
	mu      sync.Mutex
	actions map[string]*pendingAction
}

func newPendingStore() *pendingStore {
	return &pendingStore{actions: make(map[string]*pendingAction)}
}

func (p *pendingStore) add(a *pendingAction) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.actions[token] = a
	return token, nil
}

func (p *pendingStore) take(token string) (*pendingAction, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	a, ok := p.actions[token]
	if ok {
		delete(p.actions, token)
	}
	return a, ok
}

func randomToken() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
