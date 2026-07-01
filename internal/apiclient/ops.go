package apiclient

import "context"

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

// CreateTeamRequest is the body for POST /v1/teams.
type CreateTeamRequest struct {
	TeamName string `json:"team_name"`
	OrgName  string `json:"org_name"`
	Slug     string `json:"slug,omitempty"`
}

// Team is the created-team result (subset we surface).
type Team struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	APIKey string `json:"api_key"`
}

// CreateTeam provisions a team (admin-gated) and returns its key.
func (c *Client) CreateTeam(ctx context.Context, req CreateTeamRequest) (Team, error) {
	var t Team
	err := c.do(ctx, "POST", "/v1/teams", req, &t)
	return t, err
}

// CreateUserRequest is the body for POST /v1/users.
type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
	TeamID   int64  `json:"teamId"`
}

// User is the created-user result (subset we surface).
type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// CreateUser adds a user assigned to a team (admin-gated).
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (User, error) {
	var u User
	err := c.do(ctx, "POST", "/v1/users", req, &u)
	return u, err
}
