//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"os"
	"os/exec"

	"chichi/apis/transformations/micropython"
)

func main() {

	flag.Parse()

	ctx := context.Background()

	vm, err := micropython.NewVM(ctx, 10*1024*1024, os.Stdout, true)
	if err != nil {
		log.Panicf("cannot instantiate MicroPython: %s", err)
	}
	defer vm.Close()

	// Interprete a file.
	if len(flag.Args()) > 0 {
		src, err := os.ReadFile(flag.Arg(0))
		if err != nil {
			log.Panic(err)
		}
		err = vm.RunSourceCode(src)
		if err != nil {
			log.Panic(err)
		}
		return
	}

	// Run the REPL.
	disableEchoOnTTY()
	err = vm.InitREPL()
	if err != nil {
		log.Panic(err)
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			log.Panic(err)
		}
		if char == 10 {
			err = vm.REPLSendEnter()
		} else {
			err = vm.REPLSendChar(char)
		}
		if err != nil {
			log.Panic(err)
		}
	}

}

// disableEchoOnTTY disables the echo on the terminal.
func disableEchoOnTTY() {

	// See:
	// https://stackoverflow.com/questions/15159118/read-a-character-from-standard-input-in-go-without-pressing-enter

	// Disable input buffering.
	_ = exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	// Do not display entered characters on the screen.
	_ = exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

}
