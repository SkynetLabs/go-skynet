package skynet

// registry.go contains helper functions for working with the Skynet registry
// in go.

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/url"

	"gitlab.com/NebulousLabs/errors"
	"gitlab.com/SkynetLabs/skyd/node/api"
	"go.sia.tech/siad/crypto"
	"go.sia.tech/siad/modules"
	"go.sia.tech/siad/types"
)

// ErrRegistryEntryNotFound is returned when the registry entry was not found.
var ErrRegistryEntryNotFound = errors.New("registry entry was not found")

// ReadRegistry will read a registry value from Skynet, collecting both the
// data and the latest revision number.
//
// NOTE: Because this data is signed, the portal cannot provide a false
// response. The portal can however provide an out-of-date response and present
// it as the latest information. If we want to discourage this, we will need
// the portal to respond with some of the host responses, showing that the
// request was made recently and that the host has signed off that this is the
// latest data. At the very least, this would require the portal to be
// colluding with all the hosts it is providing as evidence for it to be able
// to lie to the user. We could take this even a step further by having a list
// of hosts that we want the portal to provide responses from, where we expect
// at least some percentage (such as 2/3rds) of the hosts to have responded
// successfully. This even further reduces any chance the portal has to lie to
// us.
func (sc *SkynetClient) ReadRegistry(spk types.SiaPublicKey, dataKey crypto.Hash) (data []byte, revisionNumber uint64, err error) {
	// Fetch the current revision number.
	values := url.Values{}
	values.Set("publickey", spk.String())
	values.Set("datakey", dataKey.String())
	resp, err := sc.executeRequest(
		requestOptions{
			Options: DefaultOptions("/skynet/registry"),
			method:  "GET",
			query:   values,
		},
	)
	if err != nil {
		return nil, 0, errors.AddContext(err, "unable to execute request")
	}
	defer resp.Body.Close()

	// A 404 is not an error, it just means there is no data.
	if resp.StatusCode == 404 {
		return nil, 0, ErrRegistryEntryNotFound
	}

	// Parse the response.
	if resp.ContentLength > 10e6 {
		return nil, 0, errors.New("response is larger than 10 MiB")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, errors.AddContext(err, "unable to read response body")
	}

	// The response should be json, decode it.
	var rhg api.RegistryHandlerGET
	err = json.Unmarshal(body, &rhg)
	if err != nil {
		return nil, 0, errors.AddContext(err, "unable to json parse response body")
	}
	data, err = hex.DecodeString(rhg.Data)
	if err != nil {
		return nil, 0, errors.AddContext(err, "unable to decode registry entry data")
	}
	sigBytes, err := hex.DecodeString(rhg.Signature)
	if err != nil {
		return nil, 0, errors.AddContext(err, "unable to decode registry entry signature")
	}
	var sig crypto.Signature
	copy(sig[:], sigBytes)

	// Verify the signature on the response.
	srv := modules.SignedRegistryValue{
		RegistryValue: modules.RegistryValue{
			Tweak:    dataKey,
			Data:     data,
			Revision: rhg.Revision,
			Type:     rhg.Type,
		},
		Signature: sig,
	}
	err = srv.Verify(spk.ToPublicKey())
	if err != nil {
		return nil, 0, errors.AddContext(err, "signature returned by portal does not match")
	}

	// Return the data and revision number.
	return data, rhg.Revision, nil
}

// OverwriteRegistry will overwrite an existing registry entry, replacing it
// with the provided value.
//
// WARNING: Improper use of this function can cause data loss, and has caused
// users data loss in the past. The common mistake is to first read from the
// registry entry, see that nothing is there, and then call OverwriteRegistry
// with new data. If the initial read fasely returned 404 or was subject to a
// race condition which prevented it from seeing the most recent data (both
// very common on Skynet), the resulting call to OverwriteRegistry can cause
// data loss. You should ONLY use OverwriteRegistry on registry entries that
// are not storing user data.
//
// An example of an acceptable use of this function is to deploy application
// updates to skynet. Because a newly updated application is fully independent
// of previously deployed applications, it is okay if previous data gets lost.
//
// An example of an unacceptable use of this function would be to update a list
// of a user's files. When updating the list, the new update depends on
// information previously stored in the registry (the new update is a
// combination of the previous data + the new data). Using OverwriteRegistry
// for this purpose WILL cause data loss. Instead, use a pattern where the
// registry revision number is set on the read, guaranteeing that the
// subsequent write will not destroy data in the event that the read was
// subject to a network error or network race condition.
//
// WARNING: This function does set any primary hosts, and therefore is not
// suitable for registry entries that people may wish to subscribe to. If
// people do attempt to subscribe to this registry entry, they may miss updates
// or receive updates at a significant delay.
//
// NOTE: Because this function is uploading signed data, we do not need
// substantial evidence of honesty from the portal. One thing that we could do
// to enforce a bit more accountability on the portal is require the portal to
// provide some signatures from hosts which demonstrate that the hosts received
// the update. That eliminates deceptions where the portal pretends to execute
// the registry update but actually does not. Because the data is signed, there
// is no opportunity for the portal to place false or bad data on the network.
func (sc *SkynetClient) OverwriteRegistry(secretKey crypto.SecretKey, dataKey crypto.Hash, data []byte) error {
	// Create the SiaPublicKey
	spk := types.Ed25519PublicKey(secretKey.PublicKey())

	// Fetch the revision number.
	_, revisionNumber, err := sc.ReadRegistry(spk, dataKey)
	if err != nil && err != ErrRegistryEntryNotFound {
		return errors.AddContext(err, "unable to fetch the current revision number")
	}

	// Create the request body.
	srv := modules.NewRegistryValue(dataKey, data, revisionNumber+1, modules.RegistryTypeWithoutPubkey).Sign(secretKey)
	req := api.RegistryHandlerRequestPOST{
		PublicKey: spk,
		DataKey:   srv.Tweak,
		Revision:  srv.Revision,
		Signature: srv.Signature,
		Data:      srv.Data,
		Type:      srv.Type,
	}
	// Marshal into JSON.
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return errors.AddContext(err, "unable to marshal the request")
	}
	reqBody := bytes.NewReader(reqBytes)

	// Send the request to the server.
	_, err = sc.executeRequest(
		requestOptions{
			Options: DefaultOptions("/skynet/registry"),
			method:  "POST",
			reqBody: reqBody,
		},
	)
	if err != nil {
		return errors.AddContext(err, "unable to execute request")
	}
	return nil
}
