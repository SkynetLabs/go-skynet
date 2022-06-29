package skynet

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"gitlab.com/NebulousLabs/errors"
)

var (
	PinEndpoint      = "/skynet/pin/"
	SkylinkHeaderKey = "skynet-skylink"
)

func (sc *SkynetClient) PinSkylink(skylink string) (string, error) {
	skylink = strings.TrimPrefix(skylink, "sia://")

	resp, err := sc.executeRequest(
		requestOptions{
			reqBody: nil,
			query:   url.Values{},
			Options: Options{
				EndpointPath:      PinEndpoint,
				CustomUserAgent:   sc.Options.CustomUserAgent,
				customContentType: sc.Options.customContentType,
			},
			method:    http.MethodPost,
			extraPath: skylink,
		},
	)
	if err != nil {
		return "", errors.AddContext(err, "could not execute request")
	}

	if resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("expected response status code to be %d but got %d", http.StatusNoContent, resp.StatusCode)
	}

	pinLink := resp.Header.Get(SkylinkHeaderKey)
	return pinLink, nil
}
