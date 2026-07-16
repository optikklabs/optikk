package apiclient

import (
	"context"
)

// Login authenticates and stores the returned JWT on the client.
func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	var data struct {
		AccessToken string `json:"accessToken"`
	}
	err := c.do(ctx, "POST", "/v1/auth/login",
		map[string]string{"email": email, "password": password}, &data)
	if err != nil {
		return "", err
	}
	c.token = data.AccessToken
	return data.AccessToken, nil
}

// SignupRequest is the body for POST /v1/auth/signup.
type SignupRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	Name          string `json:"name"`
	TenantName    string `json:"tenant_name"`
	AcceptedTerms bool   `json:"accepted_terms"`
}

// SignupResult is the signup response (subset we surface).
type SignupResult struct {
	AccessToken string `json:"accessToken"`
	APIKey      string `json:"api_key"`
	Message     string `json:"message"`
	Tenant      struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"tenant"`
}

// Signup self-serves an account + tenant and stores the returned JWT.
func (c *Client) Signup(ctx context.Context, req SignupRequest) (SignupResult, error) {
	var res SignupResult
	if err := c.do(ctx, "POST", "/v1/auth/signup", req, &res); err != nil {
		return SignupResult{}, err
	}
	c.token = res.AccessToken
	return res, nil
}

// OnboardingStatus is the response of GET /v1/onboarding/status (tenant-scoped).
type OnboardingStatus struct {
	ID     int64  `json:"id"`
	APIKey string `json:"api_key"`
}

// GetOnboardingStatus reports collector provisioning + first-data state.
func (c *Client) GetOnboardingStatus(ctx context.Context) (OnboardingStatus, error) {
	var st OnboardingStatus
	err := c.do(ctx, "GET", "/v1/onboarding/status", nil, &st)
	return st, err
}

// RotateAPIKey issues a fresh key for the caller's tenant (tenant-admin only).
func (c *Client) RotateAPIKey(ctx context.Context) (Tenant, error) {
	var t Tenant
	err := c.do(ctx, "POST", "/v1/settings/api-key/rotate", nil, &t)
	return t, err
}

// RevokeAPIKey disables ingest for the caller's tenant (tenant-admin only).
func (c *Client) RevokeAPIKey(ctx context.Context) (Tenant, error) {
	var t Tenant
	err := c.do(ctx, "POST", "/v1/settings/api-key/revoke", nil, &t)
	return t, err
}

// DeviceCodeResult is the response of POST /v1/auth/device/code (RFC 8628).
type DeviceCodeResult struct {
	DeviceCode string `json:"device_code"`
	UserCode   string `json:"user_code"`
	Interval   int    `json:"interval"`
	ExpiresIn  int    `json:"expires_in"`
}

// StartDeviceAuth begins the browser-handoff login and returns the code pair.
func (c *Client) StartDeviceAuth(ctx context.Context) (DeviceCodeResult, error) {
	var res DeviceCodeResult
	err := c.do(ctx, "POST", "/v1/auth/device/code", nil, &res)
	return res, err
}

// DeviceTokenResult is the poll response; Status drives the CLI loop and
// Session is populated only once Status == "complete".
type DeviceTokenResult struct {
	Status  string `json:"status"`
	Session *struct {
		AccessToken string `json:"accessToken"`
	} `json:"session"`
}

// PollDeviceToken checks whether the user has approved the device code yet.
func (c *Client) PollDeviceToken(ctx context.Context, deviceCode string) (DeviceTokenResult, error) {
	var res DeviceTokenResult
	err := c.do(ctx, "POST", "/v1/auth/device/token",
		map[string]string{"device_code": deviceCode}, &res)
	return res, err
}

// Tenant is the tenant result surfaced by settings endpoints (subset we surface).
type Tenant struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	APIKey string `json:"api_key"`
}

// CreateUserRequest is the body for POST /v1/users.
type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
	TenantID int64  `json:"tenantId"`
}

// User is the created-user result (subset we surface).
type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// CreateUser adds a user assigned to a tenant (admin-gated).
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (User, error) {
	var u User
	err := c.do(ctx, "POST", "/v1/users", req, &u)
	return u, err
}
