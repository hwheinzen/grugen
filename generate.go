// generate.go

package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
)

// V enthält alles, was der template-Mechanismus braucht.
var v Values

type Values struct {
	Generator string
	InFile    string
	OutFile   string
	Default   string
	//---
	Package string
	Import  string
	Global  string
	State   string
	Get     string
	//---
	File   Fil
	Delim  string
	RecLen string
	Record Rec
	Grus   []Gru
	UpGrus []Gru
	NoGrus bool
	//---
	GruLocs []GruLoc
	Locs    []Loc
}
type Fil struct {
	Name string
	Path string
}
type Rec struct {
	Name string
	Type string
}
type Gru struct {
	Name string // Gruppenname
	Path string // "Pfad" zur Gruppe
	Type string
}
type GruLoc struct {
	Name string
	GNam string // Gruppenname
	Path string // "Pfad" zur Gruppe
	Code string
}
type Loc struct {
	Name string
	Code string
}

// GenerateGo öffnet die Grugen-Datei mit Namen infile und gibt  sie 
// zum Lesen an die Funktion values weiter.
// Die dort gewonnenen Daten werden auf die Schablone tmpl angewendet,
// und das Ergebnis in die Ausgabedatei outFile geschrieben.
func generateGo(inFile string) (outFile string) {
	var err error

	in, err := os.Open(inFile) // Eingabedatei
	if err != nil {
		log.Fatalln(err)
	}
	defer in.Close()
	bufin := bufio.NewReader(in)

	values(bufin, inFile) // Versorgen der Values im globalen v.

	//log.Printf("Values: %#v\n", v)

	outFile = v.OutFile
	out, err := os.Create(outFile) // Ausgabedatei
	if err != nil {
		log.Fatalln(err)
	}
	defer out.Close()

	t := template.New("tmpl")
	t = t.Funcs(template.FuncMap{ // in der Schablone benötigte Funktionen
		"cat":     cat,
		"toupper": strings.ToUpper,
	})
	_, err = t.Parse(tmpl)
	if err != nil {
		log.Fatalln(err)
	}
	err = t.Execute(out, v) // Erzeugen des Go-Kodes
	if err != nil {
		log.Fatalln(err)
	}

	return
}

// Cat dient als template-Funktion.
func cat(s, t string) string {
	return s + t
}

// Values liest die Grugen-Datei und versorgt v.
func values(in *bufio.Reader, inFile string) { //
	v.Generator = pgmname
	v.InFile = inFile
	v.OutFile = "gru_" + strings.TrimSuffix(inFile, ".grugen") + "_generated.go"

	var erlaubt = make(map[string]struct{})
	erlaubt["package"] = struct{}{}
	erlaubt["import"] = struct{}{}
	erlaubt["global"] = struct{}{}
	erlaubt["state"] = struct{}{}
	erlaubt["get"] = struct{}{}

	var s, t string
	var ss, tt []string
	sp := &v.Default // zeigt jeweils auf den String der gerade aktiven Location
	var grus []Gru

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
				log.Fatalln(".gru statement not complete:", line)
			case len(grus) == 0: // Dateiebene
				grus = append(grus, Gru{Name: strings.ToLower(ss[0])})
				if len(ss) > 1 {
					t = strings.Trim(t, "\n")
					t = strings.Trim(ss[1], " ")
					tt = strings.Split(t, "=")
					switch {
					case len(tt) < 2:
						log.Fatalln("option", t, "not complete")
					case tt[0] == "limit" && strings.HasPrefix(tt[1], "'"):
						v.Delim = tt[1]
					case tt[0] == "limit":
						v.RecLen = tt[1]
						_, err := strconv.Atoi(tt[1])
						if err != nil {
							log.Fatalln("argument", tt[1], "of", tt[0], "is neither rune nor number")
						}
					default:
						log.Fatalln("unknown option:", t)
					}
				}
			default: // Gruppen- oder Satzebene
				if len(ss) == 1 { // erstes .gru bereits verarbeitet
					log.Fatalln(".gru statement not complete:", line)
				}
				grus = append(grus, Gru{Name: strings.ToLower(ss[0]), Type: ss[1]})
			}
			continue readloop // nächste Zeile
		}

		// Hier geht's weiter wenn line weder Kode noch Kommentar noch .gru ist.
		// grus enthält jetzt: File, alle Gru, Record.

		// Einmal
		if !v.NoGrus && len(v.Grus) == 0 {
			// Reihenfolge grus umdrehen
			turnGrus(grus)
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
			// Reihenfolge bei .Grus wieder umdrehen
			turnGrus(v.Grus)
		}
		if len(v.Grus) == 0 {
			v.NoGrus = true
		}

		// Einmal
		if len(v.Locs) == 0 { // != len(v.Grus)
			v.Record.Name = grus[0].Name
			v.Record.Type = grus[0].Type
			v.Locs = append(v.Locs, Loc{Name: "p_" + grus[0].Name})
			erlaubt["p_"+grus[0].Name] = struct{}{}

			last := len(grus) - 1
			v.File.Name = grus[last].Name
			v.File.Path = grus[last].Path
			v.Locs = append(v.Locs, Loc{Name: "o_" + v.File.Name})
			erlaubt["o_"+v.File.Name] = struct{}{}
			v.Locs = append(v.Locs, Loc{Name: "c_" + v.File.Name})
			erlaubt["c_"+v.File.Name] = struct{}{}
		}

		// Einmal
		if len(v.GruLocs) == 0 { // != len(v.Grus)
			for i, gru := range grus {
				if i > 0 && i < len(grus)-1 {
					v.GruLocs = append(v.GruLocs, GruLoc{
						Name: "o_" + gru.Name, GNam: gru.Name, Path: gru.Path})
					erlaubt["o_"+gru.Name] = struct{}{}
					v.GruLocs = append(v.GruLocs, GruLoc{
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
			if _, ok := erlaubt[s]; !ok { // unknown location
				log.Println("unknown location:", s)
				sp = &v.Default
				continue readloop
			}
		} else {
			log.Fatalln("unexpected statement:", s)
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

		// Für die folgenden Kodezeilen vor der nächsten .-Anweisung
		for i, loc := range v.Locs { // gru locations
			if s == loc.Name {
				sp = &v.Locs[i].Code
				continue readloop
			}
		}
		for i, loc := range v.GruLocs { // gru locations
			if s == loc.Name {
				sp = &v.GruLocs[i].Code
				continue readloop
			}
		}
		log.Fatalln("unexpected location:", s)
	}

	//log.Println("v.Locs:", v.Locs)
	//log.Println("v.GruLocs:", v.GruLocs)

	if v.Default != "" {
		log.Fatalln("code without valid location:\n" + v.Default)
	}

	if v.OutFile == "" {
		v.OutFile = "DELETE_ME.go" // für alle Fälle
	}
	if v.Package == "" {
		v.Package = "main" // default
	}
	if v.RecLen == "" && v.Delim == "" {
		v.Delim = `'\n'` // default
	}
}

// TurnGrus dreht die Reihenfolge der Elemente eines Gru-Slice.
func turnGrus(grus []Gru) {
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
