// Command encryptini encrypts a plaintext INI file for use with mitm-gui.
//
// Usage:
//
//	go run ./cmd/encryptini -password=<password> -in=<plain.ini> -out=<encrypted.enc>
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zheng-bote/go_mitm-gui/internal/crypto"
)

func main() {
	password := flag.String("password", "", "Master password for encryption")
	inFile := flag.String("in", "", "Path to plaintext INI file")
	outFile := flag.String("out", "", "Path for encrypted output file")
	flag.Parse()

	if *password == "" || *inFile == "" || *outFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: encryptini -password=<pwd> -in=<file.ini> -out=<file.enc>")
		os.Exit(1)
	}

	plaintext, err := os.ReadFile(*inFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file %q: %v\n", *inFile, err)
		os.Exit(1)
	}

	encrypted, err := crypto.EncryptFile(*password, plaintext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encryption failed: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outFile, encrypted, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file %q: %v\n", *outFile, err)
		os.Exit(1)
	}

	fmt.Printf("Encrypted %q → %q (%d bytes → %d bytes)\n",
		*inFile, *outFile, len(plaintext), len(encrypted))
}
