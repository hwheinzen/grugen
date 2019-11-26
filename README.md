# grugen
Go code generator for control break logic


## Overview
Command `grugen` is a code generator. It takes a specification file (called the Grugen file) containing a description of the sorted input data together with code snippets about what to do at the beginning and end of specific data groups. It generates a function `conbreak` featuring a simple control break logic with all the code snippets put in their appropriate places.

Put the generated code file in your project directory and use the function which is declared as follows:

`func conbreak(in *bufio.Reader, out *bufio.Writer) error`


## Acknowledgement
This pet project owes to my memory of the 4GL Delta/Gru generator my colleges and I used extensively in the 80s and 90s to generate Cobol programs. Of course that one was way more sophisticated. It might be still around somewhere.


## Purpose
Typical use of such a generator is editing sorted sequential input data for reports and/or aggregating data on any group level.


## Download
After

`$ go get github.com/hwheinzen/grugen`
  
the command grugen is at your disposal.


## Usage example
Given is the billing data of one year which is sorted by customer.

Create a Grugen file (e.g. `custtotal.grugen`) containing `.gru-` statements:
```
.gru-year
.gru-customer, string
.gru-bill, detailT
.*
.sl=global
type detailT struct {
	date string
	sum int
}
```

That tells Grugen to generate control break logic for one group level
with name `customer`. `year` will be the name of the file level,
`bill` the name of the record level.
After these `.gru` statements Grugen will know about the following
locations within the function `conbreak`:
- `o_year`	- for "open" processing of the file
- `c_year`	- for "close" processing of the file
- `o_customer`	- for "open" processing of a customer
- `c_customer`	- for "close" processing of a customer
- `p_bill`	- for processing of a single record

and the following locations within a get function: 
- `state`	- here we could ignore an input line with 'goto readagain'
- `get`		- here we need to extract information out of variable `line` and feed the variables `customerKey` and `billKey`

(The location `global` is meant for global declarations.)

Now provide code for these locations using `.sl=` statements
('sl' for 'select location'):
```
.sl=o_year
	var custTotal int
.*
.sl=o_customer
	custTotal = 0
.*
.sl=c_customer
	out.WriteString("\n" + customer + ":" + fmt.Sprint(custTotal))
.*
.sl=p_bill
	custTotal += bill.sum
.*
.sl=get
	// extract information from input line - available variable: line
	// ...
	// supply group keys and detail data
	customerKey = ...
	billKey.date = ...
	billKey.sum = ...
```
Running `grugen` on this Grugen file will generate a file `gru_custtotal_generated.go` with
the necessary Go code for printing lines to the `bufio.Writer`. Lines
will consist of customer, `:`, and the total of all the associated bills.


## More information
- Specifications for the content of the Grugen file: `doc/grugen_spec.html`.
- Examples are available in subdirectories of `examples/`.


## TODO
- automated tests for code generation
- automated tests for the generated code (function `conbreak`)
