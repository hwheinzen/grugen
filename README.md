# grugen
Go code generator for control break logic


## Overview
Command `grugen` is a code generator. It takes a specification file (called the Grugen file) containing a description of the sorted input data together with code snippets about what to do at the beginning and end of specific data groups. It generates a function `conbreak` featuring a simple control break logic with all the code snippets put in their appropriate places.

Put the generated code file in your project directory and use the function which is declared as follows:

`func conbreak(in *bufio.Reader, out *bufio.Writer) error`


## Acknowledgement
This pet project owes to my memory of the 4GL Delta/Gru generator my colleges and I used extensively in the 80s and 90s to generate Cobol programs. Of course that one was way more sophisticated. It might be still around somewhere.


## Purpose
Typical use of such a generator is editing sorted sequential input data for reporting and/or aggregating data on any group level.


## Download
Provided you have Go installed, run:

`$ go install github.com/hwheinzen/grugen@latest`

(Has been `$ go get github.com/hwheinzen/grugen` before.)


## Usage example
Consider the billing data of one year which shall be sorted by customer.
You like to get a list of customers with a total of their bills.

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

(The location `global` is for any global declarations needed.)

Now provide some code for the locations using `.sl=` statements
('sl' for 'select location'):
```
.sl=o_year
	var (
		custTotal int
		err       error
	)
.*
.sl=o_customer
	custTotal = 0
.*
.sl=c_customer
	_, err = out.WriteString("\n" + customer + ":" + fmt.Sprint(custTotal))
	if err != nil {
		return err
	}
.*
.sl=p_bill
	custTotal += bill.sum
.*
.sl=get
	// extract information from input line (variable 'line') and ...
	// ... supply group keys and detail data
	customerKey = ...
	billDetail.date = ...
	billDetail.sum = ...
```
Running `grugen` on this Grugen file will generate a file `gru_custtotal_generated.go` with
the necessary Go code for printing lines to the `bufio.Writer`. Lines
will consist of customer, `:`, and the total of all the associated bills.


## More information
- Specifications for the content of a Grugen file: `doc/grugen_spec.html`.
- There are some examples in subdirectories of `examples/`.


## TODO (maybe)
- automated tests for code generation
- automated tests for the generated code (function `conbreak`)
