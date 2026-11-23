package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrNotFound = errors.New("the hub could not find the resource")

func New(hubURL, token string) (*Client, error) {
	server, err := url.Parse(strings.TrimSpace(hubURL))
	if err != nil || (server.Scheme != "http" && server.Scheme != "https") ||
		server.Hostname() == "" || server.User != nil || server.ForceQuery ||
		server.RawQuery != "" || server.Fragment != "" {
		return nil, errors.New("use an absolute HTTP or HTTPS hub URL without credentials, query parameters, or a fragment")
	}

	server = server.JoinPath("api", "v1")
	return NewClient(server.String(), WithHTTPClient(&http.Client{
		Timeout: 90 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}), WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		return nil
	}))
}

func Decode[T any](response *http.Response, err error) (T, error) {
	var result T
	err = consume(response, err, &result)
	return result, err
}

func Delete(response *http.Response, err error) error {
	return consume(response, err, nil)
}

func consume(response *http.Response, err error, target any) error {
	if err != nil {
		// Transport errors can contain details of the request, including headers.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return errors.New("the request was canceled or timed out")
		}
		return errors.New("could not reach the OrcaCD hub; check the URL and network connection")
	}
	defer response.Body.Close()

	mediaType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	isJSON := mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
	if response.StatusCode == http.StatusNotFound && isJSON {
		return ErrNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// Validation errors may echo repository credentials or notification settings.
		detail := http.StatusText(response.StatusCode)
		switch response.StatusCode {
		case http.StatusUnauthorized:
			detail = "check that the API token is valid and has not expired"
		case http.StatusForbidden:
			detail = "check the API token scopes and its owner's permissions"
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			detail = "the hub rejected the request; check the configured values and referenced resources"
		case http.StatusConflict:
			detail = "the request conflicts with the current hub state; check for an existing object or an offline agent"
		case http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
			detail = "use the hub URL that serves the API directly"
		}
		return fmt.Errorf("the hub returned HTTP %d (%s)", response.StatusCode, detail)
	}

	if target == nil {
		if response.StatusCode != http.StatusNoContent && !isJSON {
			return errors.New("the hub returned an unexpected response to deletion")
		}
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return errors.New("the hub returned an invalid JSON response")
	}
	return nil
}
