package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/SkynetLabs/go-skynet/v2"
	"gitlab.com/NebulousLabs/fastrand"
	"gitlab.com/SkynetLabs/skyd/skymodules"
)

// main checks the args to figure out what command to run, then calls the
// corresponding command.
func main() {
	args := os.Args
	if len(args) == 2 {
		switch args[1] {
		case "generate-seed", "g", "s", "gs":
			generateAndPrintSeedPhrase()
		}
	}
	if len(args) == 3 {
		switch args[1] {
		case "upload-file", "u", "uf":
			uploadFile(args[2])
		case "upload-file-dry", "ud", "ufd":
			uploadFileDry(args[2])
		}
	}
	if len(args) > 3 {
		switch args[1] {
		case "generate-v2skylink", "p", "v2":
			generateV2SkylinkFromSeed(args[2], args[3:])
		case "upload-to-v2skylink", "u2", "u2v2", "utv", "utv2":
			uploadToV2Skylink(args[2], args[3], args[4:])
		}
	}
	printHelp()
}

// printHelp lists all of the supported commands and their functions.
func printHelp() {
	// Basic output
	fmt.Println("skynet-utils v0.0.1")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("	skynet-utils [command]")
	fmt.Println()
	fmt.Println("Available Commands:")
	// List the commands through a cleanly formatted tabwriter.
	w := tabwriter.NewWriter(os.Stdout, 2, 0, 2, ' ', 0)
	fmt.Fprintf(w, "\tgenerate-seed\tgenerates a secure seed\n")
	fmt.Fprintf(w, "\tgenerate-v2skylink [salt] [seed]\tgenerates a pubkey from a seed using the provided salt\n")
	fmt.Fprintf(w, "\tupload-file [filepath]\tuploads the provided file and returns a skylink\n")
	fmt.Fprintf(w, "\tupload-file-dry [filepath]\treturns the skylink for a file without uploading it\n")
	err := w.Flush()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println("Shortcuts:")
	// List the commands through a cleanly formatted tabwriter.
	w = tabwriter.NewWriter(os.Stdout, 2, 0, 2, ' ', 0)
	fmt.Fprintf(w, "\tgenerate-seed\t(g) (s) (gs)\n")
	fmt.Fprintf(w, "\tgenerate-v2skylink\t(p) (v2)\n")
	fmt.Fprintf(w, "\tupload-file\t(u) (uf)\n")
	fmt.Fprintf(w, "\tupload-file-dry\t(ud) (ufd)\n")
	err = w.Flush()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// generateAndPrintSeedPhrase will generate a new seed and print it.
func generateAndPrintSeedPhrase() {
	var entropy skynet.Seed
	fastrand.Read(entropy[:])
	phrase := skynet.SeedToPhrase(entropy)
	fmt.Println(phrase)
	os.Exit(0)
}

// generateV2SkylinkFromSeed will generate a V2 skylink from a seed using the
// specified salt.
func generateV2SkylinkFromSeed(salt string, phraseWords []string) {
	// Get the crypto keys.
	_, spk, dataKey := skylinkKeysFromPhraseWords(salt, phraseWords)
	skylinkV2 := skymodules.NewSkylinkV2(spk, dataKey)

	// Print the salt and seed.
	fmt.Println(skylinkV2)
	os.Exit(0)
}

// uploadFile will upload a file to the user's preferred skynet portal, which
// is detected via an environment variable. If no portal is set, siasky.net is
// used.
func uploadFile(path string) {
	client := skynet.New()
	skylink, err := client.UploadFileSecure(path)
	if err != nil {
		fmt.Println("Upload failed:", err)
		os.Exit(1)
	}
	fmt.Println(skylink)
	os.Exit(0)
}

// uploadFileDry will fetch the skylink for the provided file, without actually
// uploading the file.
func uploadFileDry(path string) {
	skylink, err := skynet.FileSkylink(path)
	if err != nil {
		fmt.Println("Unable to determine skylink:", err)
		os.Exit(1)
	}
	fmt.Println(skylink)
	os.Exit(0)
}

// uploadToV2Skylnk will upload the provided v1Skylink to the v2Skylink that
// corresponds to the provided salt and phraseWords.
func uploadToV2Skylink(v1Skylink string, salt string, phraseWords []string) {
	// Get the keys that we need.
	sk, _, dataKey := skylinkKeysFromPhraseWords(salt, phraseWords)

	// Get the raw bytes of the v1 skylink.
	var skylink skymodules.Skylink
	err := skylink.LoadString(v1Skylink)
	if err != nil {
		fmt.Println("Invalid skylink:", err)
		os.Exit(1)
	}
	linkBytes := skylink.Bytes()

	// Create a signed registry entry containing the v1skylink and upload
	// it using a portal.
	client := skynet.New()
	err = client.OverwriteRegistry(sk, dataKey, linkBytes)
	if err != nil {
		fmt.Println("Error while trying to update the registry:", err)
		os.Exit(1)
	}
	os.Exit(0)
}
