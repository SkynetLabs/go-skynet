package skynet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/NebulousLabs/errors"
)

type (
	// ErrorResponse contains the response for an error.
	ErrorResponse struct {
		// Message is the error message of the response.
		Message string `json:"message"`
	}

	// Options contains options used for connecting to a Skynet portal and
	// endpoint.
	Options struct {
		HttpClient        *http.Client
		SkynetAPIKey      string
		CustomUserAgent   string
		customContentType string
		EndpointPath      string
	}
)

const (
	// DefaultSkynetPortalURL is the default URL of the Skynet portal to use in
	// the absence of configuration.
	DefaultSkynetPortalURL = "https://siasky.net"

	// URISkynetPrefix is the URI prefix for Skynet.
	URISkynetPrefix = "sia://"
)

var (
	// ErrResponseError is the error for a response with a status code >= 400.
	ErrResponseError = errors.New("error response")
)

// DefaultOptions returns the default options with the given endpoint path.
func DefaultOptions(endpointPath string) Options {
	return Options{
		EndpointPath:    endpointPath,
		SkynetAPIKey:    "",
		CustomUserAgent: "",
		HttpClient:      http.DefaultClient,
	}
}

// DefaultPortalURL selects the default portal URL to use when initializing a
// client. May involve network queries to several candidate portals.
//
// TODO: This will be smarter. See
// https://github.com/NebulousLabs/skynet-docs/issues/21.
func DefaultPortalURL() string {
	return DefaultSkynetPortalURL
}

// ensurePrefix checks if `str` starts with `prefix` and adds it if that's not
// the case.
//
// NOTE: Taken from `skyd` - see that project for tests.
func ensurePrefix(str, prefix string) string {
	if strings.HasPrefix(str, prefix) {
		return str
	}
	return prefix + str
}

// makeResponseError makes an error given an error response.
func makeResponseError(resp *http.Response) error {
	body := &bytes.Buffer{}
	_, err := body.ReadFrom(resp.Body)
	if err != nil {
		return errors.AddContext(err, "could not read from response body")
	}

	if err = resp.Body.Close(); err != nil && err != context.Canceled {
		return errors.AddContext(err, "could not close response body")
	}

	var apiResponse ErrorResponse
	message := body.String()
	err = json.Unmarshal(body.Bytes(), &apiResponse)
	if err == nil {
		message = apiResponse.Message
	}

	context := fmt.Sprintf("%v response from %v: %v", resp.StatusCode, resp.Request.Method, message)
	return errors.AddContext(ErrResponseError, context)
}

// makeURL makes a URL from the given parts.
func makeURL(portalURL, path, extraPath string, query url.Values) string {
	url := fmt.Sprintf("%s/%s", strings.TrimRight(portalURL, "/"), strings.TrimLeft(path, "/"))
	if extraPath != "" {
		url = fmt.Sprintf("%s/%s", strings.TrimRight(url, "/"), strings.TrimLeft(extraPath, "/"))
	}
	if query == nil {
		return url
	}
	params := query.Encode()
	if params == "" {
		return url
	}
	return fmt.Sprintf("%s?%s", url, query.Encode())
}

// parseResponseBody parses the response body.
func parseResponseBody(resp *http.Response) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, errors.AddContext(err, "could not parse response body")
	}
	// TODO (jay-dee7) find a graceful way to handle this
	defer resp.Body.Close()

	// parse the response
	return buf, nil
}

// walkDirectory walks a given directory recursively, returning the paths of all
// files found.
func walkDirectory(path string) ([]string, error) {
	var files []string
	err := filepath.Walk(path, func(subpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if subpath == path {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, subpath)
		return nil
	})
	if err != nil {
		return []string{}, err
	}
	return files, nil
}
