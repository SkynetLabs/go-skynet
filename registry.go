package skynet

// registry.go contains helper functions for working with the Skynet registry
// in go.

import (
	"bytes"
	"encoding/json"

	"gitlab.com/NebulousLabs/errors"
	"gitlab.com/SkynetLabs/skyd/node/api"
	"go.sia.tech/siad/crypto"
	"go.sia.tech/siad/modules"
	"go.sia.tech/siad/types"
)

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
// registry revision number is set on the read, guaranteeing that the write
// will not cause a loss of data in the event of a network error or network
// race condition.
func (sc *SkynetClient) OverwriteRegistry(dataKey crypto.Hash, data []byte, secretKey crypto.SecretKey) error {
	// TODO: Change this to fetch a host list from the portal so that we
	// can sign all of the host pubkeys and support primary registry
	// entries.
	//
	// TODO: Extend the client to cache the list of pubkeys so we don't
	// have to fetch the full list of pubkeys each time we want to update a
	// registry entry.

	// Create the SiaPublicKey
	spk := types.Ed25519PublicKey(secretKey.PublicKey())
	srv := modules.NewRegistryValue(dataKey, data, 0, modules.RegistryTypeWithoutPubkey).Sign(secretKey)
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

	// Send the request to the server.
	_, err = sc.executeRequest(
		requestOptions{
			Options: DefaultOptions("/skynet/registry"),
			method:  "POST",
			reqBody: bytes.NewReader(reqBytes),
		},
	)
	if err != nil {
		return errors.AddContext(err, "unable to execute request")
	}
	return nil
}
