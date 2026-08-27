package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

func applyStatsParams(req *internal.Request, params *models.StatsParams) {
	if params == nil {
		return
	}
	req.WithQuery("granularity", string(params.Granularity)).WithQuery("month", params.Month)
}
