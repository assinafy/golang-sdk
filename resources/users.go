package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// UserResource exposes current-user endpoints that require client authentication
// by API key or bearer token. Its methods follow the package-level error contract.
type UserResource struct {
	http *internal.HTTPClient
}

// NewUserResource constructs a UserResource.
func NewUserResource(httpClient *internal.HTTPClient) *UserResource {
	return &UserResource{http: httpClient}
}

// GetSelf returns the authenticated user's User profile payload.
// GET /users/self.
func (r *UserResource) GetSelf(ctx context.Context) (*models.User, error) {
	var raw json.RawMessage
	if _, err := r.http.NewRequest(http.MethodGet, "/users/self").Execute(ctx, &raw); err != nil {
		return nil, err
	}
	// Accept both a direct User and the authentication-session {user, accounts}
	// shape. Only the requested User profile is returned.
	var legacy struct {
		User *models.User `json:"user"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return nil, fmt.Errorf("assinafy: decode users/self response: %w", err)
	}
	if legacy.User != nil {
		return legacy.User, nil
	}
	var out models.User
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("assinafy: decode users/self response: %w", err)
	}
	return &out, nil
}

// Stats returns DocumentStatsRow payloads summed across the authenticated user's
// current accounts and filtered by optional StatsParams. Daily
// granularity requires a YYYY-MM Month.
// GET /users/self/stats.
func (r *UserResource) Stats(ctx context.Context, params *models.StatsParams) ([]models.DocumentStatsRow, error) {
	var out []models.DocumentStatsRow
	req := r.http.NewRequest(http.MethodGet, "/users/self/stats")
	applyStatsParams(req, params)
	if _, err := req.Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetNotificationPreferences returns all NotificationPreferences for the
// authenticated user.
// GET /users/self/notification-preferences.
func (r *UserResource) GetNotificationPreferences(ctx context.Context) (models.NotificationPreferences, error) {
	var out models.NotificationPreferences
	if _, err := r.http.NewRequest(http.MethodGet, "/users/self/notification-preferences").Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateNotificationPreferences sends the supplied preference map, merges those
// keys into the authenticated user's settings, and returns the complete
// NotificationPreferences payload. Omitted keys are unchanged.
// PUT /users/self/notification-preferences.
func (r *UserResource) UpdateNotificationPreferences(ctx context.Context, changes models.NotificationPreferences) (models.NotificationPreferences, error) {
	var out models.NotificationPreferences
	if _, err := r.http.NewRequest(http.MethodPut, "/users/self/notification-preferences").WithBody(changes).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListMFAMethods returns the authenticated user's enrolled two-factor methods
// and the number of unused recovery codes.
// GET /users/self/mfa.
func (r *UserResource) ListMFAMethods(ctx context.Context) (*models.MFAStatus, error) {
	var out models.MFAStatus
	if _, err := r.http.NewRequest(http.MethodGet, "/users/self/mfa").Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StartTOTPEnrollment creates an unconfirmed authenticator method with an
// optional label and returns its TOTPEnrollment. The secret is returned only by
// this call; two-factor authentication starts after ConfirmTOTPEnrollment.
// POST /users/self/mfa/totp.
func (r *UserResource) StartTOTPEnrollment(ctx context.Context, label string) (*models.TOTPEnrollment, error) {
	body := struct {
		Label string `json:"label,omitempty"`
	}{label}
	var out models.TOTPEnrollment
	if _, err := r.http.NewRequest(http.MethodPost, "/users/self/mfa/totp").WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ConfirmTOTPEnrollment activates an enrollment with a live code from the new
// device and returns the recovery codes, which are shown only once. From then on
// every login requires a second factor. Replacing a confirmed method of the same
// type also requires Password or ReauthCode and reissues the recovery codes.
// PUT /users/self/mfa/totp/confirm.
func (r *UserResource) ConfirmTOTPEnrollment(ctx context.Context, body *models.ConfirmTOTPRequest) ([]string, error) {
	return r.recoveryCodes(ctx, http.MethodPut, "/users/self/mfa/totp/confirm", body)
}

// RegenerateRecoveryCodes issues ten new recovery codes and invalidates the
// previous set. The request must carry the current password, a live
// authenticator code or an unused recovery code, which is then consumed.
// POST /users/self/mfa/recovery-codes.
func (r *UserResource) RegenerateRecoveryCodes(ctx context.Context, body *models.MFAReauthRequest) ([]string, error) {
	return r.recoveryCodes(ctx, http.MethodPost, "/users/self/mfa/recovery-codes", body)
}

// RemoveMFAMethod removes an enrolled method after the same re-authentication
// as RegenerateRecoveryCodes, and reports whether two-factor authentication is
// still enabled. Removing the last method also discards the recovery codes.
// DELETE /users/self/mfa/{method_id}.
func (r *UserResource) RemoveMFAMethod(ctx context.Context, methodID string, body *models.MFAReauthRequest) (bool, error) {
	var out struct {
		IsMFAEnabled bool `json:"is_mfa_enabled"`
	}
	if _, err := r.http.NewRequest(http.MethodDelete, "/users/self/mfa/"+url.PathEscape(methodID)).WithBody(body).Execute(ctx, &out); err != nil {
		return false, err
	}
	return out.IsMFAEnabled, nil
}

func (r *UserResource) recoveryCodes(ctx context.Context, method, path string, body any) ([]string, error) {
	var out struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	if _, err := r.http.NewRequest(method, path).WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out.RecoveryCodes, nil
}
