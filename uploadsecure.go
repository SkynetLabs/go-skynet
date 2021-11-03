package skynet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gitlab.com/NebulousLabs/encoding"
	"gitlab.com/NebulousLabs/errors"
	"gitlab.com/SkynetLabs/skyd/node/api"
	"gitlab.com/SkynetLabs/skyd/skymodules"
	"go.sia.tech/siad/crypto"
	"go.sia.tech/siad/persist"
)

// uploadsecure.go performs uploads to skynet portals via the /skynet/restore
// endpoint. This is because the /skynet/restore endpoint will upload a file in
// a bit-perfect manner, meaning the skylink can reliably be computed prior to
// the upload even across different software versions. This allows us to verify
// that the file uploaded to Skynet is the exact file we wanted to upload,
// instead of trusting the skylink that the portal returns.

// uploadFileSecure is the implementation for UploadFileSecure, except there is
// dryRun support to get just the skylink and not actually upload anything.
func (sc *SkynetClient) uploadFileSecure(path string, dryRun bool) (skylinkStr string, err error) {
	// Clean the path.
	path = filepath.Clean(path)

	// Open the file.
	file, err := os.Open(path)
	if err != nil {
		return "", errors.AddContext(err, "unable to open file")
	}

	// Get the filesize. Abort if it's larger than 4 million bytes, to
	// leave room for metadata.
	info, err := file.Stat()
	if err != nil {
		return "", errors.AddContext(err, "unable to fetch filesize")
	}
	if info.Size() > 4e6 {
		return "", errors.New("cannot perform SecureUploadFile with files larger than 4e6 bytes")
	}

	// Pull the full file into memory to build the base sector.
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", errors.AddContext(err, "unable to read full file")
	}

	// Get the metadata bytes.
	metadata := skymodules.SkyfileMetadata{
		Filename: info.Name(),
		Mode:     info.Mode(),
		Length:   uint64(info.Size()),
	}
	// Double check that the metadata is valid.
	err = skymodules.ValidateSkyfileMetadata(metadata)
	if err != nil {
		return "", errors.AddContext(err, "metadata is returning as invalid")
	}
	metadataBytes, err := skymodules.SkyfileMetadataBytes(metadata)
	if err != nil {
		return "", errors.AddContext(err, "could not fetch the metadata bytes")
	}

	// Create the layout bytes.
	layoutBytes := skymodules.NewSkyfileLayoutNoFanout(uint64(len(fileBytes)), uint64(len(metadataBytes)), crypto.TypePlain).Encode()

	// Build the base sector. The 'nil' is for the fanout bytes. Since the
	// full file fits into the base sector, there is no fanout and
	// therefore no fanout bytes.
	baseSector, fetchSize, _ := skymodules.BuildBaseSector(layoutBytes, nil, metadataBytes, fileBytes)

	// Pull the skylink from the base sector.
	baseSectorRoot := crypto.MerkleRoot(baseSector)
	skylink, err := skymodules.NewSkylinkV1(baseSectorRoot, 0, fetchSize)
	if err != nil {
		return "", errors.AddContext(err, "unable to create the skylink from the base sector")
	}

	// If this is a dryRun, return the skylink without uploading the file.
	if dryRun {
		return skylink.String(), nil
	}

	// Build the backup reader.
	encodedHeader := encoding.Marshal(skymodules.SkyfileBackupHeader{
		Metadata: persist.Metadata{
			Header:  skymodules.MetadataHeader,
			Version: skymodules.MetadataVersion,
		},
		Skylink: skylink.String(),
	})
	// NOTE: 92 is a constant in the skymodules package that is not
	// exported. It's the 'backupHeaderSize' constant.
	if len(encodedHeader) > 92 {
		return "", errors.New("backup header exceeded the maximum size")
	}
	reqBody := bytes.NewReader(append(encodedHeader, baseSector...))
	fmt.Println(len(baseSector))

	// Perform the request.
	resp, err := sc.executeRequest(
		requestOptions{
			Options: DefaultOptions("/skynet/restore"),
			method:  "POST",
			reqBody: reqBody,
		})
	if err != nil {
		return "", errors.AddContext(err, "upload request failed")
	}

	// Parse the response.
	respBody, err := parseResponseBody(resp)
	if err != nil {
		return "", errors.AddContext(err, "unable to parse upload response")
	}
	fmt.Println(string(respBody.Bytes()))
	var srp api.SkynetRestorePOST
	err = json.Unmarshal(respBody.Bytes(), &srp)
	if err != nil {
		return "", errors.AddContext(err, "unable to unmarshal upload response")
	}

	// Verify that the response matches the skylink we computed locally.
	if srp.Skylink != skylink.String() {
		return "", errors.New("portal skylink differs from locally computed skylink")
	}
	return skylink.String(), nil
}

// FileSkylink will return the skylink that would be produced from uploading
// the file using UploadFileSecure.
func FileSkylink(path string) (skylinkStr string, err error) {
	// Create a nil SkynetClient to call the function. This won't cause a
	// panic because the funciton aborts early due to the dryRun flag.
	var sc *SkynetClient
	return sc.uploadFileSecure(path, true)
}

// UploadFileSecure takes a filepath as input and will upload the file securely
// to a Skynet portal. SecureUpload differs from other upload types in that it
// will compute the skylink prior to uploading and then use the /skynet/restore
// endpoint to ensure that the file gets uploaded correctly.
//
// NOTE: Currently UploadFileSecure only supports files that can fit inside of
// a base sector. As need arises (and... need may never arise), we will extend
// this endpoint to support full sized files, which will include use of the TUS
// protocol.
func (sc *SkynetClient) UploadFileSecure(path string) (skylinkStr string, err error) {
	return sc.uploadFileSecure(path, false)
}
