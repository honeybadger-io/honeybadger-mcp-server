package hbmcp

import "strings"

// CredentialKind is which sort of Bearer credential a request carries.
//
// v3 accepts three, and they differ in what this server can verify locally: an
// OAuth access token is a signed JWT, while the scoped API tokens are opaque
// strings that only the API can judge.
type CredentialKind string

const (
	// KindOAuth is an OAuth access token issued by the authorization server.
	// Signed, so its issuer, expiry, and audience are all checkable here.
	KindOAuth CredentialKind = "oauth"

	// KindUserToken is a scoped API token belonging to a person.
	KindUserToken CredentialKind = "user"

	// KindAccountToken is a scoped API token belonging to an account.
	KindAccountToken CredentialKind = "account"

	// KindUnknown is anything else, which this server refuses.
	KindUnknown CredentialKind = ""
)

// Credential prefixes, as documented by the API's bearer_auth scheme.
const (
	oauthPrefix        = "hbo_"
	userTokenPrefix    = "hbt_"
	accountTokenPrefix = "hba_"
)

// Verifiable reports whether this kind can be validated without calling the API.
// Only OAuth tokens can: the scoped tokens carry no structure to check.
func (k CredentialKind) Verifiable() bool {
	return k == KindOAuth
}

// ClassifyCredential identifies a Bearer credential by its prefix.
//
// Prefix matching is a routing decision, not an authorization one — it picks how
// to handle the credential, and the API remains the only thing that decides
// whether it is genuine.
func ClassifyCredential(raw string) CredentialKind {
	switch {
	case strings.HasPrefix(raw, oauthPrefix):
		return KindOAuth
	case strings.HasPrefix(raw, userTokenPrefix):
		return KindUserToken
	case strings.HasPrefix(raw, accountTokenPrefix):
		return KindAccountToken
	default:
		return KindUnknown
	}
}
