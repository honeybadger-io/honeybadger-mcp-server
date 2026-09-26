// Package confirmtoken mints and verifies short-lived tokens proving that a
// destructive action was previewed before it runs.
package confirmtoken

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const TTL = 10 * time.Minute

// Lets a replica whose clock trails the issuer's accept a fresh token.
const clockSkew = 30 * time.Second

// Signer's key must be unknown to callers: a key they hold, such as their
// own bearer token, would let them mint a token without the preview.
type Signer struct {
	key []byte
}

func New(key []byte) *Signer {
	return &Signer{key: key}
}

// NewRandom suits a single process that both mints and verifies.
func NewRandom() *Signer {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return New(key)
}

// Mint returns a token for caller performing action on ids, valid for TTL.
func (s *Signer) Mint(caller, action string, ids []any, now time.Time) string {
	exp := strconv.FormatInt(now.Add(TTL).Unix(), 36)
	return exp + "." + s.mac(caller, action, ids, exp)
}

func (s *Signer) Valid(token, caller, action string, ids []any, now time.Time) bool {
	exp, mac, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	unix, err := strconv.ParseInt(exp, 36, 64)
	if err != nil || now.Unix() >= unix || unix > now.Add(TTL+clockSkew).Unix() {
		return false
	}
	return hmac.Equal([]byte(mac), []byte(s.mac(caller, action, ids, exp)))
}

// JSON encoding keeps field boundaries unambiguous, so ids [1, 23] and
// [12, 3] never share a MAC.
func (s *Signer) mac(caller, action string, ids []any, exp string) string {
	msg, _ := json.Marshal(append([]any{action, exp, caller}, ids...))
	m := hmac.New(sha256.New, s.key)
	m.Write(msg)
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil)[:16])
}
