package main

/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"flag"
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/version"
)

func main() {
	var get_version bool
	var encrypt_from_stdin bool
	var decrypt_from_stdin bool
	var paas_name string
	var encrypt_from_file string
	var keyfile string
	var generate bool
	flag.BoolVar(&get_version, "version", false, "Print version and quit")
	flag.StringVar(&paas_name, "paas-name", "", "The name of the PaaS object to encrypt data for")
	flag.StringVar(&encrypt_from_file, "encrypt-from-file", "", "The path to the file to be encrypted")
	flag.BoolVar(&encrypt_from_stdin, "encrypt-from-stdin", false, "Encrypt data from stdin")
	flag.BoolVar(&decrypt_from_stdin, "decrypt-from-stdin", false, "Decrypt data read from stdin")
	flag.StringVar(&keyfile, "key", "", "The path to the private or public key used for de-/encryption")
	flag.BoolVar(&generate, "generate", false, "Generate new encryption keys")
	flag.Parse()
	if decrypt_from_stdin {
		crypt.DecryptFromStdin(keyfile, paas_name)
	} else if encrypt_from_stdin {
		crypt.EncryptFromStdin(keyfile, paas_name)
	} else if encrypt_from_file != "" {
		crypt.EncryptFile(keyfile, paas_name, encrypt_from_file)
	} else if generate {
		crypt.GenerateKeyPair()
	} else if get_version {
		fmt.Printf("opr-paas version %s", version.PAAS_VERSION)
	} else {
		fmt.Println("Pease supply arguments")
	}
}
