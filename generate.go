// generate.go

package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
)

type valT struct {
	Generator string
	InFile    string
	OutFile   string

	Package string
	Import  string
	Global  string
	State   string
	Get     string

	File   filType
	Delim  string
	RecLen string
	Record recType
	Grus   []gruType
	UpGrus []gruType

	GruLocs []gruLocType
	Locs    []locType
}
type filType struct {
	Name string
	Path string
}
type recType struct {
	Name string
	Type string
}
type gruType struct {
	Name string // Gruppenname
	Path string // "Pfad" zur Gruppe
	Type string
}
type gruLocType struct {
	Name string
	GNam string // Gruppenname
	Path string // "Pfad" zur Gruppe
	Code string
}
type locType struct {
	Name string
	Code string
}

// Die Daten in v werden auf die Schablone tmpl angewendet,
// und das Ergebnis in die Ausgabedatei v.OutFile geschrieben.
func generate(v valT) error {

	out, err := os.Create(v.OutFile) // Ausgabedatei
	if err != nil {
		return err
	}
	defer out.Close()

	t := template.New("tmpl")
	t = t.Funcs(template.FuncMap{ // in der Schablone benötigte Funktionen
		"cat":     cat,
		"toupper": strings.ToUpper,
	})
	_, err = t.Parse(tmpl)
	if err != nil {
		return err
	}
	err = t.Execute(out, v) // Erzeugen des Go-Kodes
	if err != nil {
		return err
	}

	return nil
}

// Cat dient als template-Funktion.
func cat(s, t string) string {
	return s + t
}

// Values liest die Grugen-Datei und versorgt v.
func values(filename string) (valT, error) { //

	// V enthält alles, was der template-Mechanismus später braucht
	var v valT

	inFile, err := os.Open(filename) // Eingabedatei
	if err != nil {
		return v, err
	}
	defer inFile.Close()
	in := bufio.NewReader(inFile)

	v.Generator = pgmname
	v.InFile = filename
	v.OutFile = "gru_" + strings.TrimSuffix(filename, ".grugen") + "_generated.go"

	var erlaubt = make(map[string]struct{})
	erlaubt["package"] = struct{}{}
	erlaubt["import"] = struct{}{}
	erlaubt["global"] = struct{}{}
	erlaubt["state"] = struct{}{}
	erlaubt["get"] = struct{}{}

	var s, t string
	var ss, tt []string
	var defLoc string // für Kode ohne gültige Zuweisung
	sp := &defLoc     // zeigt jeweils auf den String der gerade aktiven location
	var grus []gruType

readloop:
	for line, eod := nextLine(in); !eod; line, eod = nextLine(in) {

		// Kommentar
		if strings.HasPrefix(line, ".*") {
			continue readloop // nächste Zeile
		}

		// Kode
		if line[0] != '.' {
			*sp += line // Zeile an aktive Location anhängen
			continue readloop // nächste Zeile
		}

		s = strings.ToLower(line)

		// .gru
		if strings.HasPrefix(s, ".gru-") {
			s = line[5:] // wg. Groß-Klein-Schreibung wieder Eingabezeile
			s = strings.TrimSuffix(s, "\n")
			ss = strings.Split(s, ",")
			switch {
			case len(ss) == 0: // hier fehlt alles
				return v, errors.New(".gru statement not complete: " + line)
			case len(grus) == 0: // Dateiebene
				grus = append(grus, gruType{Name: strings.ToLower(ss[0])})
				if len(ss) > 1 {
					t = strings.Trim(t, "\n")
					t = strings.Trim(ss[1], " ")
					tt = strings.Split(t, "=")
					switch {
					case len(tt) < 2:
						return v, errors.New("option " + t + " not complete")
					case tt[0] == "limit" && strings.HasPrefix(tt[1], "'"):
						v.Delim = tt[1]
					case tt[0] == "limit":
						v.RecLen = tt[1]
						_, err := strconv.Atoi(tt[1])
						if err != nil {
							return v, errors.New("argument " + tt[1] + " of " + tt[0] +  " is neither rune nor number")
						}
					default:
						return v, errors.New("unknown option: " + t)
					}
				}
			default: // Gruppen- oder Satzebene
				if len(ss) == 1 { // erstes .gru bereits verarbeitet
					return v, errors.New(".gru statement not complete: " + line)
				}
				grus = append(grus, gruType{Name: strings.ToLower(ss[0]), Type: ss[1]})
			}
			continue readloop // nächste Zeile
		}

		// Hier geht's weiter wenn line weder Kode noch Kommentar noch .gru ist.
		// grus enthält jetzt: File, alle Gru, Record.

		if len(v.Grus) == 0 { // nur einmal
			flip(grus) // Reihenfolge grus umdrehen
			// .Path erzeugen
			prevGru := ""
			for i, gru := range grus {
				if prevGru != "" && i > 0 {
					grus[i].Path = prevGru + "." + gru.Name
				}
				if prevGru == "" {
					prevGru = gru.Name
					continue
				}
				prevGru = prevGru + "." + gru.Name
			}
			// Nur Gruppenebenen -> .Grus .UpGrus
			for i, gru := range grus {
				if i == 0 || i == len(grus)-1 {
					continue
				}
				v.Grus = append(v.Grus, gru)
				v.UpGrus = append(v.UpGrus, gru)
			}
			flip(v.Grus) // Reihenfolge bei .Grus wieder umdrehen
		}

		if len(v.Locs) == 0 { // nur einmal
			v.Record.Name = grus[0].Name
			v.Record.Type = grus[0].Type
			v.Locs = append(v.Locs, locType{Name: "p_" + grus[0].Name})
			erlaubt["p_"+grus[0].Name] = struct{}{}

			last := len(grus) - 1
			v.File.Name = grus[last].Name
			v.File.Path = grus[last].Path
			v.Locs = append(v.Locs, locType{Name: "o_" + v.File.Name})
			erlaubt["o_"+v.File.Name] = struct{}{}
			v.Locs = append(v.Locs, locType{Name: "c_" + v.File.Name})
			erlaubt["c_"+v.File.Name] = struct{}{}
		}

		if len(v.GruLocs) == 0 { // nur einmal
			for i, gru := range grus {
				if i > 0 && i < len(grus)-1 {
					v.GruLocs = append(v.GruLocs, gruLocType{
						Name: "o_" + gru.Name, GNam: gru.Name, Path: gru.Path})
					erlaubt["o_"+gru.Name] = struct{}{}
					v.GruLocs = append(v.GruLocs, gruLocType{
						Name: "c_" + gru.Name, GNam: gru.Name, Path: gru.Path})
					erlaubt["c_"+gru.Name] = struct{}{}
				}
			}
		}

		s = strings.ToLower(line)

		// .sl - select location
		if strings.HasPrefix(s, ".sl ") || strings.HasPrefix(s, ".sl=") {
			s = s[4:]
			s = strings.TrimSuffix(s, "\n")
			if _, ok := erlaubt[s]; !ok {
				//return v, errors.New("unknown location: " + s)
				log.Println("unknown location:", s)
				sp = &defLoc
				continue readloop
			}
		} else {
			return v, errors.New("unexpected statement: " + s)
		}
		if strings.HasPrefix(s, "package") {
			sp = &v.Package
			continue readloop
		}
		if strings.HasPrefix(s, "import") {
			sp = &v.Import
			continue readloop
		}
		if strings.HasPrefix(s, "global") {
			sp = &v.Global
			continue readloop
		}
		if strings.HasPrefix(s, "state") {
			sp = &v.State
			continue readloop
		}
		if strings.HasPrefix(s, "get") {
			sp = &v.Get
			continue readloop
		}
		for i, loc := range v.GruLocs { // gru locations
			if s == loc.Name {
				sp = &v.GruLocs[i].Code
				continue readloop
			}
		}
		for i, loc := range v.Locs { // file/record locations
			if s == loc.Name {
				sp = &v.Locs[i].Code
				continue readloop
			}
		}
		return v, errors.New("unexpected location: " + s)
	}

	if v.Package == "" {
		v.Package = "main" // default
	}
	if v.RecLen == "" && v.Delim == "" {
		v.Delim = `'\n'` // default
	}

	if defLoc != "" {
		return v, errors.New("code without valid location:\n" + defLoc)
	}

	return v, nil
}

// Flip dreht die Reihenfolge der Elemente eines gruType-Slice.
func flip(grus []gruType) {
	for i, gru := range grus {
		if i+1 > len(grus)/2 {
			break
		}
		grus[i] = grus[len(grus)-1-i]
		grus[len(grus)-1-i] = gru
	}
}

var inEOF bool

func nextLine(in *bufio.Reader) (line string, eod bool) {
	if inEOF { // vorher: EOF + Daten
		eod = true
		return
	}

	line, err := in.ReadString('\n') // Lesen
	if err != nil && err != io.EOF { // echter Lesefehler
		log.Fatalln(err)
	}

	if err == io.EOF && len(line) == 0 { // EOF + keine Daten
		eod = true
		return
	}
	if err == io.EOF && len(line) > 0 { // EOF + Daten
		inEOF = true
		line += "\n" // letzte Zeile war ohne \n
	}
	return
}
