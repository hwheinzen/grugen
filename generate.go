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

// ReadValues ruft values.
func readValues(filename string) (v valT, err error) { //

	inFile, err := os.Open(filename) // Eingabedatei
	if err != nil {
		return v, err
	}
	defer inFile.Close()
	in := bufio.NewReader(inFile)

	v, err = values(in) // Werte zum Versorgen der Schablone in generate
	if err != nil {
		return v, err
	}

	v.InFile = filename
	v.OutFile = "gru_" + strings.TrimSuffix(filename, ".grugen") + "_generated.go"

	return v, nil
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
func values(in *bufio.Reader) (valT, error) { //

	var v valT // alles, was der template-Mechanismus später braucht

	var locs = make(map[string]struct{})
	var valid = struct{}{}
	locs["package"] = valid
	locs["import"] = valid
	locs["global"] = valid
	locs["state"] = valid
	locs["get"] = valid

	var s, t string
	var ss, tt []string
	var defLoc string // für Kode ohne gültige Zuweisung
	locp := &defLoc   // zeigt jeweils auf den String der gerade aktiven location
	var grus []gruType

readloop:
	for line, eod := nextLine(in); !eod; line, eod = nextLine(in) {

		// Kommentar
		if strings.HasPrefix(line, ".*") {
			continue readloop // nächste Zeile
		}

		// Kode
		if line[0] != '.' {
			*locp += line // Zeile an aktive Location anhängen
			continue readloop // nächste Zeile
		}

		// .gru
		s = strings.ToLower(line)
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

		// Hier geht's weiter wenn line weder Kode noch Kommentar noch .gru ist,
		// d.h. wahrscheinlich .sl= .
		// grus enthält jetzt: File, alle Gru, Record.
		if len(v.Grus) == 0 { // nur einmal
			makePaths(&v, grus)
		}
		if len(v.Locs) == 0 { // nur einmal
			v.File.Name = grus[0].Name
			v.File.Path = grus[0].Path
			v.Locs = append(v.Locs, locType{Name: "o_" + v.File.Name})
			locs["o_"+v.File.Name] = valid
			v.Locs = append(v.Locs, locType{Name: "c_" + v.File.Name})
			locs["c_"+v.File.Name] = valid

			last := len(grus) - 1
			v.Record.Name = grus[last].Name
			v.Record.Type = grus[last].Type
			v.Locs = append(v.Locs, locType{Name: "p_" + v.Record.Name})
			locs["p_"+v.Record.Name] = valid
		}
		if len(v.GruLocs) == 0 { // nur einmal
			for i, gru := range grus {
				if i > 0 && i < len(grus)-1 {
					v.GruLocs = append(v.GruLocs, gruLocType{
						Name: "o_" + gru.Name, GNam: gru.Name, Path: gru.Path})
					locs["o_"+gru.Name] = valid
					v.GruLocs = append(v.GruLocs, gruLocType{
						Name: "c_" + gru.Name, GNam: gru.Name, Path: gru.Path})
					locs["c_"+gru.Name] = valid
				}
			}
		}

		// .sl - select location - Ab hier geht's um die Locations.
		s = strings.ToLower(line)
		if strings.HasPrefix(s, ".sl ") || strings.HasPrefix(s, ".sl=") {
			s = s[4:]
			s = strings.TrimSuffix(s, "\n")
			if _, ok := locs[s]; !ok {
				//return v, errors.New("unknown location: " + s)
				log.Println("unknown location:", s)
				locp = &defLoc
				continue readloop
			}
		} else {
			return v, errors.New("unexpected statement: " + s)
		}
		if strings.HasPrefix(s, "package") {
			locp = &v.Package
			continue readloop
		}
		if strings.HasPrefix(s, "import") {
			locp = &v.Import
			continue readloop
		}
		if strings.HasPrefix(s, "global") {
			locp = &v.Global
			continue readloop
		}
		if strings.HasPrefix(s, "state") {
			locp = &v.State
			continue readloop
		}
		if strings.HasPrefix(s, "get") {
			locp = &v.Get
			continue readloop
		}
		for i, loc := range v.GruLocs { // gru locations
			if s == loc.Name {
				locp = &v.GruLocs[i].Code
				continue readloop
			}
		}
		for i, loc := range v.Locs { // file/record locations
			if s == loc.Name {
				locp = &v.Locs[i].Code
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
		return v, errors.New("code without locs location:\n" + defLoc)
	}

	return v, nil
}

// MakePaths ergänzt Path in grus und versorgt v.Grus und v.UpGrus.
func makePaths(v *valT, grus []gruType) {
	flip(grus) // Reihenfolge grus umdrehen

	prevGru := ""
	for i, gru := range grus {
		if prevGru != "" && i > 0 {
			grus[i].Path = prevGru + "." + gru.Name // Path versorgen
		}

		if prevGru == "" { // Start
			prevGru = gru.Name
			continue
		}
		prevGru = prevGru + "." + gru.Name // Name anhängen
	}

	for i, gru := range grus {
		if i == 0 || i == len(grus)-1 { // Datei- und Satzebene nicht
			continue
		}
		v.Grus = append(v.Grus, gru)
		v.UpGrus = append(v.UpGrus, gru)
	}
	flip(v.Grus) // Reihenfolge bei .Grus umdrehen

	flip(grus)   // Reihenfolge bei grus wieder zurück
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
