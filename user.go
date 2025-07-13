package dotenv

import (
	"context"
	"net/http"
	"time"
)

// UserService handles user operations
type UserService struct {
	client *Client
}

// User represents an authenticated user
type User struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	IsVerified bool      `json:"is_verified"`
}

// UserOrganization represents a user's organization membership
type UserOrganization struct {
	ID       string    `json:"id"`
	ULID     string    `json:"ulid"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// UserResponse represents the user info API response
type UserResponse struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Name       string    `json:"name"`
			Email      string    `json:"email"`
			CreatedAt  time.Time `json:"created_at"`
			UpdatedAt  time.Time `json:"updated_at"`
			IsVerified bool      `json:"is_verified"`
		} `json:"attributes"`
		Relationships struct {
			Organizations struct {
				Data []struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"organizations"`
		} `json:"relationships"`
	} `json:"data"`
	Included []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			ULID string `json:"ulid"`
			Name string `json:"name"`
			Slug string `json:"slug"`
			Role string `json:"role"`
		} `json:"attributes"`
	} `json:"included"`
}

// GetAuthenticatedUser retrieves information about the authenticated user
func (s *UserService) GetAuthenticatedUser(ctx context.Context) (*User, []*UserOrganization, *http.Response, error) {
	u := "/api/v1/user"

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	var userResp UserResponse
	resp, err := s.client.Do(ctx, req, &userResp)
	if err != nil {
		return nil, nil, resp, err
	}

	// Parse user data
	user := &User{
		ID:         userResp.Data.ID,
		Name:       userResp.Data.Attributes.Name,
		Email:      userResp.Data.Attributes.Email,
		CreatedAt:  userResp.Data.Attributes.CreatedAt,
		UpdatedAt:  userResp.Data.Attributes.UpdatedAt,
		IsVerified: userResp.Data.Attributes.IsVerified,
	}

	// Parse organizations from included data
	organizations := make([]*UserOrganization, 0)
	for _, inc := range userResp.Included {
		if inc.Type == "organizations" {
			org := &UserOrganization{
				ID:   inc.ID,
				ULID: inc.Attributes.ULID,
				Name: inc.Attributes.Name,
				Slug: inc.Attributes.Slug,
				Role: inc.Attributes.Role,
			}
			organizations = append(organizations, org)
		}
	}

	return user, organizations, resp, nil
}
