package dotenv

import (
	"context"
	"fmt"
	"net/http"
)

// SecretVersionsService handles secret backup/version history operations.
//
// Versions are immutable snapshots of a level's encrypted blob, captured on every
// write. They may be encrypted under older keys (see EncryptionKeyVersion); the
// caller decrypts client-side, supplying an old key when needed.
type SecretVersionsService struct {
	client *Client
}

type versionListEnvelope struct {
	Data []struct {
		ID         string        `json:"id"`
		Attributes SecretVersion `json:"attributes"`
	} `json:"data"`
	Meta *Meta `json:"meta"`
}

type versionContentEnvelope struct {
	Data struct {
		ID         string                `json:"id"`
		Attributes SecretVersion         `json:"attributes"`
		Content    string                `json:"content"`
		Key        *EncryptionKeyVersion `json:"key"`
	} `json:"data"`
}

// List returns the version history for a level (deepest of project/target/environment).
func (s *SecretVersionsService) List(ctx context.Context, project, target, environment string, opts *VersionListOptions) ([]*SecretVersion, *Meta, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = withDeepestResource(ctx, project, target, environment)
	u := fmt.Sprintf("/api/v1/%s/secrets/versions", s.client.organization)

	body := map[string]interface{}{"project": project}
	if target != "" {
		body["target"] = target
	}
	if environment != "" {
		body["environment"] = environment
	}
	if opts != nil {
		if opts.Page > 0 {
			body["page"] = opts.Page
		}
		if opts.PerPage > 0 {
			body["per_page"] = opts.PerPage
		}
	}

	req, err := s.client.NewRequest(ctx, "POST", u, body)
	if err != nil {
		return nil, nil, nil, err
	}

	var env versionListEnvelope
	resp, err := s.client.Do(ctx, req, &env)
	if err != nil {
		return nil, nil, resp, err
	}

	versions := make([]*SecretVersion, 0, len(env.Data))
	for _, d := range env.Data {
		v := d.Attributes
		v.ID = d.ID
		versions = append(versions, &v)
	}

	return versions, env.Meta, resp, nil
}

// Get fetches a single version including its encrypted content and key descriptor.
func (s *SecretVersionsService) Get(ctx context.Context, versionID string) (*SecretVersionContent, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/secrets/versions/%s", s.client.organization, versionID)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var env versionContentEnvelope
	resp, err := s.client.Do(ctx, req, &env)
	if err != nil {
		return nil, resp, err
	}

	out := &SecretVersionContent{
		SecretVersion: env.Data.Attributes,
		Content:       env.Data.Content,
		Key:           env.Data.Key,
	}
	out.ID = env.Data.ID

	return out, resp, nil
}

// Restore restores a version (append-only). For client-managed old-key versions
// pass req with the re-encrypted Content and current-key KeyProof; otherwise nil.
func (s *SecretVersionsService) Restore(ctx context.Context, versionID string, req *RestoreVersionRequest) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/secrets/versions/%s/restore", s.client.organization, versionID)

	var body interface{}
	if req != nil {
		body = req
	}

	httpReq, err := s.client.NewRequest(ctx, "POST", u, body)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, httpReq, nil)
}

// Delete removes a single version.
func (s *SecretVersionsService) Delete(ctx context.Context, versionID string) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/secrets/versions/%s", s.client.organization, versionID)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// PurgeHistory deletes version history for a level or the whole project,
// returning the number of versions purged.
func (s *SecretVersionsService) PurgeHistory(ctx context.Context, req PurgeHistoryRequest) (int, *http.Response, error) {
	if s.client.organization == "" {
		return 0, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/secrets/versions/purge", s.client.organization)

	httpReq, err := s.client.NewRequest(ctx, "POST", u, req)
	if err != nil {
		return 0, nil, err
	}

	var env struct {
		Data struct {
			PurgedCount int `json:"purged_count"`
		} `json:"data"`
	}
	resp, err := s.client.Do(ctx, httpReq, &env)
	if err != nil {
		return 0, resp, err
	}

	return env.Data.PurgedCount, resp, nil
}
