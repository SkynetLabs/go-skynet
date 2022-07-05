package main

import (
	"fmt"
	"os"

	"github.com/SkynetLabs/go-skynet/v2"
	"go.sia.tech/siad/crypto"
	"go.sia.tech/siad/types"
)

// skylinkKeysFromPhraseWords returns the entropy for a given seed and salt.
func skylinkKeysFromPhraseWords(salt string, phraseWords []string) (sk crypto.SecretKey, spk types.SiaPublicKey, dataKey crypto.Hash) {
	// Turn the phrase words in to a phrase.
	var phrase string
	for i, word := range phraseWords {
		phrase += word
		if i != len(phraseWords)-1 {
			phrase += " "
		}
	}

	// Turn the phrase into entropy.
	seed, err := skynet.PhraseToSeed(phrase)
	if err != nil {
		fmt.Println("Invalid seed provided:", err)
		os.Exit(1)
	}
	// Use the salt to deterministically generate entropy for this specific
	// V2 link. Add some pepper to the salt to minimize footgun potential.
	//
	// We want the data key to appear random, so we are going to hash a
	// value deterministically to get that as well. We are going to use a
	// different pepper but the same salt to get the data key.
	saltedSeed := "v2SkylinkFromSeed" + salt + string(seed[:])
	dataKeyBase := "v2SkylinkFromSeedDataKey" + salt + string(seed[:])
	entropy := crypto.HashObject(saltedSeed)
	dataKey = crypto.HashObject(dataKeyBase)

	// Get the actual crypto keys.
	sk, pk := crypto.GenerateKeyPairDeterministic(entropy)
	spk = types.Ed25519PublicKey(pk)

	// Return the secret key and the SiaPublicKey.
	return sk, spk, dataKey
}
