package hbmcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// withAccount runs an operation, and if v3 refuses because "me" is ambiguous,
// finds out which account the credential belongs to and runs it again.
//
// The answer comes from the request's cached introspection when there is one, so
// recovery normally costs no extra API call at all.
//
// Stateless by construction: the resolved id lives in this call and is gone when
// it returns. Nothing is cached on the client or carried between requests, which
// is what a hosted server needs — but it also means a credential covering several
// accounts pays the extra round trip every time. Passing account_id explicitly avoids
// it.
//
// The whole operation is retried rather than a single request, so a paginated
// walk resolves once instead of per page.
func withAccount[T any](
	ctx context.Context,
	client *apiv3.Client,
	requested string,
	call func(accountID string) (T, error),
) (T, error) {
	result, err := call(requested) // "" lets apiv3 send the `me` sentinel
	if !errors.Is(err, apiv3.ErrAmbiguousAccount) {
		return result, err
	}

	// The http middleware has usually already introspected this credential and
	// cached the result, so prefer that over asking again — otherwise every
	// ambiguous-account call costs an extra /v3/token round trip that the cache
	// exists to avoid.
	if info := TokenInfoFromContext(ctx); info != nil && info.AccountID != "" {
		return call(info.AccountID)
	}

	// No cached description: stdio has no middleware to attach one.
	info, introspectErr := client.Tokens.Get(ctx)
	if introspectErr != nil || info.AccountID == "" {
		// Report what the caller asked about, not the recovery attempt.
		return result, err
	}
	return call(info.AccountID)
}

// inAccount turns an optional account id into request options.
func inAccount(accountID string) []apiv3.Option {
	if accountID == "" {
		return nil
	}
	return []apiv3.Option{apiv3.InAccount(accountID)}
}

// listAllInAccount is inAccount for the ListAll methods, which take the narrower
// option type.
func listAllInAccount(accountID string) []apiv3.ListAllOption {
	if accountID == "" {
		return nil
	}
	return []apiv3.ListAllOption{apiv3.InAccount(accountID)}
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

// projectSettingsNotInV3 are project fields v2 accepted that v3's write schema
// does not declare.
//
// A field absent from the schema does not exist in the generated request type, so
// it cannot be sent. Rejecting the request is the honest response: accepting it
// and dropping the field would report success for a change that never happened.
var projectSettingsNotInV3 = []string{
	"resolve_errors_on_deploy",
	"disable_public_links",
	"user_url",
	"source_url",
	"purge_days",
	"user_search_field",
}

// rejectUnsupportedProjectSettings returns an error message when a request
// carries project settings v3 cannot express, or "" when it is safe to proceed.
// rejectUnsupportedProjectSettings returns an error message when a request
// carries project settings v3 cannot express, or "" when it is safe to proceed.
//
// Presence is what matters, not truthiness. Checking values would let
// disable_public_links:false through — a perfectly valid request to turn the
// setting off — and then drop it, which is exactly the silent no-op this guard
// exists to prevent.
func rejectUnsupportedProjectSettings(req mcp.CallToolRequest) string {
	return rejectUnsupported(req, projectSettingsNotInV3, "writing a project",
		"change them in the Honeybadger UI, and retry without them")
}

// requireProjectAndFault reads the two ids every fault tool needs.
func requireProjectAndFault(req mcp.CallToolRequest) (projectID, faultID, errMsg string) {
	projectID = req.GetString("project_id", "")
	if projectID == "" {
		return "", "", "project_id is required"
	}
	faultID = req.GetString("fault_id", "")
	if faultID == "" {
		return "", "", "fault_id is required"
	}
	return projectID, faultID, ""
}

// optionalBool reads a boolean argument that may be absent.
//
// Read from the raw arguments rather than through the typed getter, which coerces
// invalid input — null becomes false — and would turn a malformed request into a
// silent state change.
func optionalBool(args map[string]any, name string) (value, present bool, err error) {
	raw, ok := args[name]
	if !ok {
		return false, false, nil
	}
	v, ok := raw.(bool)
	if !ok {
		return false, false, fmt.Errorf("%s must be a boolean", name)
	}
	return v, true, nil
}

// rejectUnsupported refuses a request carrying parameters v3 cannot express.
//
// Silently ignoring them is the one option not on the table: a caller who filters
// or sets something and gets a success back has been told the wrong thing.
func rejectUnsupported(req mcp.CallToolRequest, fields []string, action, advice string) string {
	args := req.GetArguments()
	var present []string
	for _, field := range fields {
		if _, ok := args[field]; ok {
			present = append(present, field)
		}
	}
	if len(present) == 0 {
		return ""
	}
	return fmt.Sprintf("The v3 API does not support %s when %s, so they would be ignored. %s.",
		strings.Join(present, ", "), action, advice)
}
