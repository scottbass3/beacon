package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	IssuedAt     string `json:"issued_at"`
}

func decodeTokenResponse(resp *http.Response) (string, string, time.Time, error) {
	var payload tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", time.Time{}, err
	}
	token := firstNonEmptyToken(payload.IDToken, payload.AccessToken, payload.Token)
	refresh := payload.RefreshToken
	expiry := time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	if payload.ExpiresIn == 0 {
		expiry = time.Now().Add(5 * time.Minute)
	}
	return token, refresh, expiry, nil
}

func parseBearerChallenge(value string) (realm, service, scope string, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(value), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", "", "", false
	}

	for _, segment := range strings.Split(parts[1], ",") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		kv := strings.SplitN(segment, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
		switch key {
		case "realm":
			realm = val
		case "service":
			service = val
		case "scope":
			scope = val
		}
	}
	if realm == "" {
		return "", "", "", false
	}
	return realm, service, scope, true
}

func fetchBearerToken(ctx context.Context, client *http.Client, logger RequestLogger, realm, service, scope string) (string, time.Time, error) {
	tokenURL, err := url.Parse(realm)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid token realm: %w", err)
	}
	query := tokenURL.Query()
	if service != "" {
		query.Set("service", service)
	}
	if scope != "" {
		query.Set("scope", scope)
	}
	tokenURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	logRequestWithLogger(logger, req, resp)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", time.Time{}, fmt.Errorf("token request failed: %s", resp.Status)
	}

	token, _, expiry, err := decodeTokenResponse(resp)
	if err != nil {
		return "", time.Time{}, err
	}
	if token == "" {
		return "", time.Time{}, fmt.Errorf("token response missing token")
	}
	return token, expiry, nil
}

func logRequestWithLogger(logger RequestLogger, req *http.Request, resp *http.Response) {
	if logger == nil {
		return
	}
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	logger(RequestLog{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: cloneHeader(req.Header),
		Status:  status,
	})
}

func firstNonEmptyToken(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
