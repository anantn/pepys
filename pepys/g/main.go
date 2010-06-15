package main

import "os"
import "fmt"
import "flag"
import "json"
import "sort"
import "strconv"
import "io/ioutil"

// Language
import "./@LANG@"

type ParsedOps map[string][]map[string]string

// Parses JSON description file and returns parsed data structure and codes
func parse(bytes []byte) ParsedOps {
	messages := make(ParsedOps, 1)
	err := json.Unmarshal(bytes, &messages)
	if err != nil {
		fmt.Printf("JSON parse error: %v\n", err)
		os.Exit(1)
	}
	return messages
}

// Generate description from ParsedOps
func arrange(desc ParsedOps) @LANG@.Description {
	nops := len(desc)
	clist := make([]int, nops)
	codes := make(map[int]string, nops)
	sdesc := make(@LANG@.Description, nops)
	
	// Extract codes for each message type
	i := 0
	for op, spec := range desc {
		for _, atype := range spec {
			for argname, argtype := range atype {
				if argname == "code" {
					clist[i], _ = strconv.Atoi(argtype)
					codes[clist[i]] = op
					// Remove code from desc
					atype[argname] = "0", false
					
					i = i + 1
				}
			}
		}
	}
	
	// Populate description
	i = 0
	sort.SortInts(clist)
	
	for _, code := range clist {
		sdesc[i].Code = code
		sdesc[i].Name = codes[code]
		sdesc[i].Args = desc[codes[code]]
		i = i + 1
	}
	
	return sdesc
}

func main() {
	args := flag.Args()
	json, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Printf("Could not read %v\n", args[0])
		os.Exit(1)
	}
	
	var fils []string
	desc := arrange(parse(json))
	
	rel := false
	if len(args) == 2 {
		rel = true
	}
	
	fils = @LANG@.Generate(desc, "@LANG@", rel)
	for _, f := range fils {
		fmt.Printf("Generated %s\n", f)
	}
}
