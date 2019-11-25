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

	out := generateGo(in)

	cmd := exec.Command("gofmt", "-w", out)

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err := cmd.Run()
	if err != nil {
		log.Println(stderr.String())
		log.Fatalln(err)
	}

	return
}
