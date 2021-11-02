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

// UpdateRegistry will update a registry entry to a new value. It will
// overwrite whatever is already in the registry.
//
// WARNING: Improper use of this function can cause data loss, and has caused
// users data loss in the past. A common mistake is to read the registry entry
// first, see that it is blank (due to either a 404, a timeout, or an
// unreliable read), and then call UpdateRegistry with some new data.
// UpdateRegistry then will see the real data (because it does not 404 when it
// tries to read), set the revision number correctly, and then obliterate the
// user's existing data.
//
// WARNING (continued): If you are using UpdateRegistry, ensure that you do not
// care whether anything is already written in the registry. If you are reading
// the registry entry first to check that it is empty, you need to ensure that
// you save the revision number and use that revision number to perform any
// updates. As this function does not accept a revision number as an input, you
// cannot use the registry safely in that scenario.
//
// TODO: I am contemplating not supporting this function at all, and instead
// jumping straight to a more sophisticated read/write system like getsetjson
// to eliminate the footgun. I am certain that the warnings will be ignored and
// that user data will be lost.
//
// TODO: I'm not sure this is the simplest/cleanest API for managing the
// dataKey. An improvement is probably going to require adding some helper
// functions for computing the v2skylinks.
func (sc *SkynetClient) UpdateRegistry(dataKey crypto.Hash, data []byte, secretKey crypto.SecretKey) error {
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
		DataKey: srv.Tweak,
		Revision: srv.Revision,
		Signature: srv.Signature,
		Data: srv.Data,
		Type: srv.Type,
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
			method: "POST",
			reqBody: bytes.NewReader(reqBytes),
		},
	)
	if err != nil {
		return errors.AddContext(err, "unable to execute request")
	}
	return nil
}
