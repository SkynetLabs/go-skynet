package skynet

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gitlab.com/NebulousLabs/errors"
)

type (
	Metadata struct {
		ContentType   string
		Etag          string
		Skylink       string
		ContentLength int
	}
)

// Metadata downloads metadata from the given skylink.
func (sc *SkynetClient) Metadata(skylink string, opts MetadataOptions) (*Metadata, error) {
	skylink = strings.TrimLeft(skylink, "sia://")

	config := requestOptions{
		query:     map[string][]string{},
		Options:   opts.Options,
		method:    http.MethodHead,
		extraPath: skylink,
	}

	resp, err := sc.executeRequest(config)
	if err != nil {
		fmt.Println("SKYNET: failed to execute req: ", err.Error())
		return nil, errors.AddContext(err, "failed to execute request")
	}

	// Metadata API's biggest use case is for checking content-length and using is for concurrent downloads
	// If contentLength is missing, it's sort of is equivalent of an error
	if resp.Header.Get("content-length") == "" {
		fmt.Println("SKYNET: failed to get header: ", err.Error())
		return nil, fmt.Errorf("error retrieving metadata for skylink: %s - ContentLength is absent", skylink)
	}

	var metadata Metadata
	contentLength, err := strconv.ParseInt(resp.Header.Get("content-length"), 10, 64)
	if err != nil {
		return nil, err
	}

	metadata.ContentLength = int(contentLength)
	metadata.Skylink = resp.Header.Get("skynet-skylink")
	metadata.ContentType = resp.Header.Get("content-type")
	metadata.Etag = resp.Header.Get("etag")

	return &metadata, nil
}
