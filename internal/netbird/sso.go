package netbird

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type SSOClient interface {
	BeginSSO(context.Context, string) (SSOLogin, error)
	WaitSSO(context.Context, string) error
}
type SSOSession struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type SSOService struct {
	client   SSOClient
	mu       sync.Mutex
	sessions map[string]SSOSession
}

func NewSSOService(c SSOClient) *SSOService {
	return &SSOService{client: c, sessions: map[string]SSOSession{}}
}
func (s *SSOService) Start(ctx context.Context, managementURL string) (SSOSession, string, error) {
	login, err := s.client.BeginSSO(ctx, managementURL)
	if err != nil {
		return SSOSession{}, "", err
	}
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		return SSOSession{}, "", err
	}
	v := SSOSession{ID: hex.EncodeToString(b), Status: "pending", ExpiresAt: time.Now().Add(10 * time.Minute)}
	s.mu.Lock()
	s.sessions[v.ID] = v
	s.mu.Unlock()
	go func() {
		c, cancel := context.WithDeadline(context.Background(), v.ExpiresAt)
		defer cancel()
		err := s.client.WaitSSO(c, login.UserCode)
		s.mu.Lock()
		defer s.mu.Unlock()
		x := s.sessions[v.ID]
		if err != nil {
			x.Status = "failed"
		} else {
			x.Status = "completed"
		}
		s.sessions[v.ID] = x
	}()
	return v, login.VerificationURI, nil
}
func (s *SSOService) Status(id string) (SSOSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.sessions[id]
	if !ok {
		return SSOSession{}, errors.New("SSO session not found")
	}
	if time.Now().After(v.ExpiresAt) && v.Status == "pending" {
		v.Status = "expired"
		s.sessions[id] = v
	}
	return v, nil
}
