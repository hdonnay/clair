// +build tools

// Embed_openapi is a script to take the OpenAPI YAML file, turn it into a JSON
// document, and embed it into a source file for easy deployment.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

func main() {
	// expects to be ran from /httptransport directory
	inFile := flag.String("in", "../openapi.yaml", "input YAML file")
	sourceFile := flag.String("src", "./discoveryhandler_gen.go", "the source file to embed openapi spec into")
	outFile := flag.String("out", "./discoveryhandler_gen.go", "output go file")
	flag.Parse()

	inF, err := os.Open(*inFile)
	if err != nil {
		log.Fatal(err)
	}
	defer inF.Close()
	tmp := map[interface{}]interface{}{}
	if err := yaml.NewDecoder(inF).Decode(&tmp); err != nil {
		log.Fatal(err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, *sourceFile, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	embed, err := json.Marshal(convert(tmp))
	if err != nil {
		log.Fatal(err)
	}
	ck := sha256.Sum256(embed)

	for i, decl := range f.Decls {
		constDecl := decl.(*ast.GenDecl)
		for _, spec := range constDecl.Specs {
			vs := spec.(*ast.ValueSpec)
			name := vs.Names[i].Name
			value := vs.Values[i]
			switch name {
			case "_openapiJSON":
				value.(*ast.BasicLit).Value = "`" + string(embed) + "`"
			case "_openapiJSONEtag":
				value.(*ast.BasicLit).Value = "`" + fmt.Sprintf("%x", ck) + "`"
			}
		}
	}

	outF, err := os.OpenFile(*outFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer outF.Close()
	err = format.Node(outF, fset, f)
	if err != nil {
		log.Fatal(err)
	}
}

// Convert yoinked from:
// https://stackoverflow.com/questions/40737122/convert-yaml-to-json-without-struct/40737676#40737676
func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[fmt.Sprint(k)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}
