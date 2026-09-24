package confirmtoken

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestValid(t *testing.T) {
	s := New([]byte("server-secret"))
	now := time.Now()
	ids := []any{123}
	good := s.Mint("alice", "delete_project", ids, now)

	if !s.Valid(good, "alice", "delete_project", ids, now) {
		t.Fatal("freshly minted token rejected")
	}
	if !s.Valid(good, "alice", "delete_project", ids, now.Add(TTL-time.Second)) {
		t.Fatal("token rejected before it expired")
	}

	exp, mac, _ := strings.Cut(good, ".")
	farFuture := strconv.FormatInt(now.Add(365*24*time.Hour).Unix(), 36)
	rejects := map[string]struct {
		signer *Signer
		token  string
		caller string
		action string
		ids    []any
		now    time.Time
	}{
		"expired":       {s, good, "alice", "delete_project", ids, now.Add(TTL)},
		"other action":  {s, good, "alice", "delete_dashboard", ids, now},
		"other id":      {s, good, "alice", "delete_project", []any{124}, now},
		"id type":       {s, good, "alice", "delete_project", []any{"123"}, now},
		"other caller":  {s, good, "bob", "delete_project", ids, now},
		"other key":     {New([]byte("caller-bearer-token")), good, "alice", "delete_project", ids, now},
		"swapped exp":   {s, farFuture + "." + mac, "alice", "delete_project", ids, now},
		"exp over TTL":  {s, farFuture + "." + s.mac("alice", "delete_project", ids, farFuture), "alice", "delete_project", ids, now},
		"tampered mac":  {s, exp + "." + strings.Repeat("A", len(mac)), "alice", "delete_project", ids, now},
		"no separator":  {s, exp + mac, "alice", "delete_project", ids, now},
		"garbage":       {s, "not-a-token", "alice", "delete_project", ids, now},
		"empty":         {s, "", "alice", "delete_project", ids, now},
		"id split move": {s, s.Mint("alice", "delete_fault_comment", []any{1, 23, 4}, now), "alice", "delete_fault_comment", []any{12, 3, 4}, now},
	}
	for name, tc := range rejects {
		t.Run(name, func(t *testing.T) {
			if tc.signer.Valid(tc.token, tc.caller, tc.action, tc.ids, tc.now) {
				t.Error("token accepted")
			}
		})
	}
}

func TestValidToleratesClockSkew(t *testing.T) {
	s := New([]byte("server-secret"))
	issued := time.Now()
	token := s.Mint("alice", "delete_project", []any{1}, issued)

	if !s.Valid(token, "alice", "delete_project", []any{1}, issued.Add(-clockSkew+time.Second)) {
		t.Error("rejected by a verifier whose clock trails the issuer within the skew allowance")
	}
	if s.Valid(token, "alice", "delete_project", []any{1}, issued.Add(-clockSkew-time.Second)) {
		t.Error("accepted by a verifier trailing beyond the skew allowance")
	}
}

func TestNewRandomKeysDiffer(t *testing.T) {
	now := time.Now()
	token := NewRandom().Mint("", "delete_project", []any{1}, now)
	if NewRandom().Valid(token, "", "delete_project", []any{1}, now) {
		t.Error("token from one random signer accepted by another")
	}
}
