// Example: tabs

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
)

const pgmname = "tabs"

func main() {
	file := args()
	fin, err := os.Open(file)
	if err != nil {
		log.Fatalln(err)
	}
	defer fin.Close()
	in := bufio.NewReader(fin)

	fout, err := os.Create("list.txt")
	if err != nil {
		log.Fatalln(err)
	}
	defer fout.Close()
	out := bufio.NewWriter(fout)

	err = conbreak(in, out)
	if err != nil {
		log.Fatalln(err)
	}
	return
}

// Args liest das erste String-Arguments und prüft, ob es ein gültiger
// Dateiname ist.
func args() (filename string) {
	var help bool
	flag.BoolVar(&help, "help", false, "usage information")
	flag.Usage = usage
	flag.Parse()

	if help {
		usage()
		os.Exit(0)
	}

	if len(flag.Args()) < 1 {
		log.Fatalln("1 argument wanted -", len(flag.Args()), "arguments found")
	}

	filename = flag.Args()[0]
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
	fmt.Fprintln(os.Stderr, "usage:", pgmname, "-help | input_filename")
	flag.PrintDefaults()
}
