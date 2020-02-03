// main.go

package main

import (
	"bytes"
	"log"
	"os/exec"
)

const pgmname = "grugen"

// Kommando grugen liest die Grugen-Steuerdatei, deren Name als Argument
// mitgegeben wurde, wendet die dort hinterlegten Anweisungen auf eine
// text/template-Schablone an und gibt das Ergebnis als generierte
// und mit gofmt formatierte Go-Quelldatei aus.
//
// Grugen eignet sich zum Einsatz mit //go:generate.
func main() {
	in := args()

	v, err := values(in) // Werte zum Versorgen der Schablone in generate
	if err != nil {
		log.Fatalln(err)
	}

	err = generate(v)
	if err != nil {
		log.Fatalln(err)
	}

	cmd := exec.Command("gofmt", "-w", v.OutFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err = cmd.Run()
	if err != nil {
		log.Println(stderr.String())
		log.Fatalln(err)
	}

	return
}
