package main

const tmpl = `// ------------- DO NOT EDIT -------------
// THIS FILE HAS BEEN GENERATED BY {{.Generator}}
// (PROBABLY VIA go generate)
// USING {{.InFile}}.

// ---------- PACKAGE
package {{.Package}}
// ------ end PACKAGE

import (
	"bufio"
	"fmt"
	"io"
{{- if not .NoGrus}}
	"reflect"
{{- end}}

// ---------- IMPORT
{{.Import -}}
// ------ end IMPORT
)

// {{.Record.Name}}T declares the type of the variable which controls the logic.
// It concatenates the group keys in reverse order and contains record details too.
// if the programmer so wishes.
type {{.Record.Name}}T struct {
	value {{.Record.Type}}
{{- range $i, $gru := .UpGrus }}
	{{$gru.Name}} {{$gru.Name}}T
	}
	type {{$gru.Name}}T struct {
		value {{$gru.Type}}
{{- end}}
	{{.File.Name}} {{.File.Name}}T
}
type {{.File.Name}}T struct {
	eod bool
}

type gruT struct {
	{{.Record.Name}} {{.Record.Name}}T
	old{{.Record.Name}} {{.Record.Name}}T
	eof bool
}

// The following types are used to check if input is sorted.
{{- range $i, $gru := .Grus }}
		type {{$gru.Name}}Map map[{{$gru.Name}}T]struct{}
{{- end}}

// ---------- GLOBAL
{{.Global -}}
// ------ end GLOBAL

// Conbreak contains the control break logic and includes the code snippets
// that the programmer provided for the following locations.
//   o/c_<file level name>
//   o/c_<group level name>
//   p_<record level name>
// If the input data is not properly sorted conbreak stops working and 
// returns an error value.
func conbreak(in *bufio.Reader, out *bufio.Writer) error {
	var gru gruT
	{{- range $i, $gru := .Grus }}
		gru_{{$gru.Name}}s := make({{$gru.Name}}Map, 1000)
	{{- end}}

	gru_err := {{.File.Name}}Get(in, &gru) // initial read
	if gru_err != nil {
		return gru_err
	}
	// group keys are now available

	// fill and use local variables
	{{- range .Grus}}
		{{.Name}}    := gru.{{.Path}}.value
		_ = {{.Name}}
	{{- end}}
	{{.Record.Name}}    := gru.{{.Record.Name}}.value
	_ = {{.Record.Name}}
	
	{{- $locs := .Locs}}
	{{- $record := .Record.Name}}

	{{with $loc := cat "o_" .File.Name}}
	{{- range $locs }}
		{{- if eq $loc .Name}}
			{{- if ne .Code ""}}
	// ---------- {{toupper .Name}}
				{{.Code -}}
	// ------ end {{toupper .Name}}
			{{- else}}
	// ---------- {{toupper .Name}} empty
			{{- end}}
		{{- end}}
	{{- end}}
	{{- end}}

	for !gru.{{.File.Path}}.eod { // until end-of-data

	{{- $grulocs := .GruLocs}}
	{{- range $i, $gru := .Grus}}
			
		// check sorting (not necessarily ordering!)
		if _, exists := gru_{{$gru.Name}}s[gru.{{$gru.Path}}]; exists {
			return fmt.Errorf("input is not sorted:\n %v\nis in wrong position", gru.{{$record}})
		}
		gru_{{$gru.Name}}s[gru.{{$gru.Path}}] = struct{}{} // remember
			
		gru.old{{$gru.Path}}.value = gru.{{$gru.Path}}.value // fill compare value
		{{$gru.Name}} = gru.{{.Path}}.value // fill local variable
			
		{{with $loc := cat "o_" $gru.Name}}
		{{- range $grulocs }}
			{{- if eq $loc .Name}}
				{{- if ne .Code ""}}
		// ---------- {{toupper .Name}}
					{{.Code -}}
		// ------ end {{toupper .Name}}
				{{- else}}
		// ---------- {{toupper .Name}} empty
				{{- end}}
			{{- end}}
		{{- end}}
		{{- end}}

		for reflect.DeepEqual(gru.{{$gru.Path}}, gru.old{{$gru.Path}}) {

	{{- end}}

			gru.old{{.Record.Name}}.value = gru.{{.Record.Name}}.value // fill compare value
			{{.Record.Name}}    = gru.{{.Record.Name}}.value // fill local variable

			{{with $loc := cat "p_" .Record.Name}}
			{{- range $locs }}
				{{- if eq $loc .Name}}
					{{- if ne .Code ""}}
			// ---------- {{toupper .Name}}
						{{.Code -}}
			// ------ end {{toupper .Name}}
					{{- else}}
			// ---------- {{toupper .Name}} empty
					{{- end}}
				{{- end}}
			{{- end}}
			{{- end}}

			gru_err = {{.File.Name}}Get(in, &gru) // next read
			if gru_err != nil {
				return gru_err
			}

	{{- range $i, $gru := .UpGrus}}
		}
			
		{{with $loc := cat "c_" $gru.Name}}
		{{- range $grulocs }}
			{{- if eq $loc .Name}}
				{{- if ne .Code ""}}
		// ---------- {{toupper .Name}}
					{{.Code -}}
		// ------ end {{toupper .Name}}
				{{- else}}
		// ---------- {{toupper .Name}} empty
				{{- end}}
			{{- end}}
		{{- end}}
		{{- end}}
	{{- end}}

	}
	{{with $loc := cat "c_" .File.Name}}
	{{- range $locs }}
		{{- if eq $loc .Name}}
			{{- if ne .Code ""}}
	// ---------- {{toupper .Name}}
				{{.Code -}}
	// ------ end {{toupper .Name}}
			{{- else}}
	// ---------- {{toupper .Name}} empty
			{{- end}}
		{{- end}}
	{{- end}}
	{{- end}}
	
	out.Flush()
	return nil
}

// {{.File.Name}}Get reads the next line from in and feeds the control variable
// gru.
// If an error occurs {{.File.Name}}Get stops working and returns an error value.
func {{.File.Name}}Get(in *bufio.Reader, gru *gruT) error {
	if gru.eof { // earlier: EOF + data
		gru.{{.File.Path}}.eod = true
		return nil
	}

	goto readAgain // must be used at least once
readAgain:
	{{- if ne .RecLen ""}}
		rec := make([]byte, {{.RecLen}})
		n, gru_err := io.ReadFull(in, rec)
		if gru_err != nil && 
			gru_err != io.EOF && 
			gru_err != io.ErrUnexpectedEOF { // read error
			return gru_err
		}
		if gru_err == io.EOF || 
			(gru_err == io.ErrUnexpectedEOF && rec[0] == '\n') { // EOF + no data
			gru.{{.File.Path}}.eod = true
			return nil
		}
		if (gru_err == io.EOF || gru_err == io.ErrUnexpectedEOF) && 
			len(rec) > 1 { // EOF + data
			gru.eof = true
		}

		line := string(rec[:n]) // ReadFull delivered n bytes

		if len(line) != {{.RecLen}} {
			log.Fatalln("wrong lenght record:\n" + line)
		}
	{{- else}}
		line, gru_err := in.ReadString({{.Delim}}) // read
		//log.Println("len(line):", len(line), "\n line:", line, "\n gru_err :", gru_err) // TEST
		if gru_err != nil && gru_err != io.EOF { // read error
			//log.Println("gru_err != nil && gru_err != io.EOF // read error") // TEST
			return gru_err
		}
		if gru_err == io.EOF && len(line) == 0 { // EOF + no data
			//log.Println("gru_err == io.EOF && len(line) == 0 // EOF + no data") // TEST
			gru.{{.File.Path}}.eod = true
			return nil
		}
		if gru_err == io.EOF && len(line) > 0 { // EOF + data
			//log.Println("gru_err == io.EOF && len(line) > 0 // EOF + data") // TEST
			gru.eof = true
		}
		if line[len(line)-1] == {{.Delim}} ||
			line[len(line)-1] == '\n'  {
			line = line[:len(line)-1] // data without the newline/delimiter rune
		}
	{{- end}}

	//log.Println("len(line):", len(line), "\n line:", line) // TEST

	{{if ne .State ""}}
		// ---------- {{toupper "State"}}
		// ----------    variable 'line' is available
		// ----------    to ignore line use 'goto readAgain'
		{{.State -}}
		// ------ end {{toupper "State"}}
	{{- else}}
		// ---------- {{toupper "State"}} empty
	{{- end}}

	{{range .Grus}}
		{{.Name}}Key    := gru.{{.Path}}.value
	{{- end}}
	{{.Record.Name}}Key := gru.{{.Record.Name}}.value


	//log.Println("keys  vor Get:", nameKey, groupKey, recKey.euro) // TEST

	{{if ne .Get ""}}
		// ---------- {{toupper "Get"}}
		// ----------    variable 'line' is available
		// ----------    to ignore line use 'goto readAgain'
		// ----------    new group keys MUST be filled here!
		{{.Get -}}
		// ------ end {{toupper "Get"}}
	{{- else}}
		// ---------- {{toupper "Get"}} empty
	{{- end}}

	//log.Println("keys nach Get:", nameKey, groupKey, recKey.euro) // TEST

	{{range .Grus}}
		gru.{{.Path}}.value    = {{.Name}}Key
	{{- end}}
	gru.{{.Record.Name}}.value    = {{.Record.Name}}Key
	return nil
}

// THIS FILE HAS BEEN GENERATED BY {{.Generator}}
// (PROBABLY VIA go generate)
// USING {{.InFile}}.
// ------------- DO NOT EDIT -------------`
