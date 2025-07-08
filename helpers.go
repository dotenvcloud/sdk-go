package dotenv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

// addOptions adds query parameters to a URL
func addOptions(s string, opts *ListOptions) string {
	if opts == nil {
		return s
	}

	u, _ := url.Parse(s)
	q := u.Query()

	if opts.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	if opts.PerPage > 0 {
		q.Set("per_page", fmt.Sprintf("%d", opts.PerPage))
	}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}

	for k, v := range opts.Filter {
		q.Set(fmt.Sprintf("filter[%s]", k), v)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// mapToStruct converts a map to a struct using JSON marshal/unmarshal
func mapToStruct(m map[string]interface{}, v interface{}) error {
	// Handle time fields specially
	rv := reflect.ValueOf(v).Elem()
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}

		// Get the JSON field name
		jsonFieldName := strings.Split(jsonTag, ",")[0]

		// Check if this is a time field
		if field.Type == reflect.TypeOf(time.Time{}) {
			if val, ok := m[jsonFieldName]; ok {
				if strVal, ok := val.(string); ok {
					// Try parsing different time formats
					var t time.Time
					var err error

					// Try RFC3339
					t, err = time.Parse(time.RFC3339, strVal)
					if err != nil {
						// Try without timezone
						t, err = time.Parse("2006-01-02T15:04:05", strVal)
					}
					if err != nil {
						// Try date only
						t, err = time.Parse("2006-01-02", strVal)
					}

					if err == nil {
						rv.Field(i).Set(reflect.ValueOf(t))
						// Remove from map so json unmarshal doesn't try to process it
						delete(m, jsonFieldName)
					}
				}
			}
		}
	}

	// Use json marshal/unmarshal for the rest
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}

// parseJSONResponse parses JSON from an HTTP response
func parseJSONResponse(resp *http.Response, v interface{}) error {
	if resp == nil || resp.Body == nil {
		return fmt.Errorf("nil response or body")
	}
	defer resp.Body.Close()
	
	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(v)
}
