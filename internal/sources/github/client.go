package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse[T any] struct {
	Data   T              `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

type graphqlError struct {
	Message string `json:"message"`
}

// doGraphQL executes a GraphQL query against the given endpoint and decodes the
// response data into T. Variables may contain nil values for optional fields
// (serialized as JSON null).
func doGraphQL[T any](ctx context.Context, token, endpoint, query string, variables map[string]any) (T, error) {
	var zero T

	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return zero, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("create github request: %w", err)
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return zero, fmt.Errorf("do github request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("github api: unexpected status %d", resp.StatusCode)
	}

	var gr graphqlResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return zero, fmt.Errorf("decode github response: %w", err)
	}
	if len(gr.Errors) > 0 {
		return zero, fmt.Errorf("github graphql: %s", gr.Errors[0].Message)
	}

	return gr.Data, nil
}
