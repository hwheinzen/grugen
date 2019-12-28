// args.go

package main

import (
	"flag"
	"log"
	"os"
	"strings"
)

// Args liest das erste String-Arguments und prüft, ob es ein gültiger
// Dateiname ist.
func args() (filename string) {
	var help bool
	flag.BoolVar(&help, "help", false, "usage information")
	flag.Usage = usage

	flag.Parse()

	if help {
		usage()
		os.Exit(0) // not an error
	}

	if len(flag.Args()) != 1 {
		log.Println("1 argument wanted -", len(flag.Args()), "arguments found")
		usage()
		os.Exit(1)
	}

	filename = flag.Args()[0]
	if !strings.HasSuffix(filename, ".grugen") {
		log.Println("input_filename lacks suffix .grugen")
		usage()
		os.Exit(1)
	}

	fi, err := os.Stat(filename)
	if err != nil {
		log.Fatalln(err)
	}
	if fi.IsDir() {
		log.Fatalln(fi.Name(), "is not a file")
	}
	if fi.Size() == 0 {
		log.Fatalln(fi.Name(), "is empty")
	}
	return
}

func usage() {
	log.Println("usage:", pgmname, "-help | input_filename")
	flag.PrintDefaults()
	log.Println("  input_filename needs suffix .grugen")
}
