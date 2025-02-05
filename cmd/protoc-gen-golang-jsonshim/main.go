// Copyright 2019 Istio Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"path"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			generateFile(gen, f)
		}
		return nil
	})
}

func generateFile(gen *protogen.Plugin, file *protogen.File) {
	filename := file.GeneratedFilenamePrefix + "_json.gen.go"
	p := gen.NewGeneratedFile(filename, file.GoImportPath)

	p.P("// Code generated by protoc-gen-jsonshim. DO NOT EDIT.")
	p.P("package ", file.GoPackageName)
	var process func([]*protogen.Message)

	marshalerName := FileName(file) + "Marshaler"
	unmarshalerName := FileName(file) + "Unmarshaler"

	process = func(messages []*protogen.Message) {
		for _, message := range messages {
			// skip maps in protos.
			if message.Desc.Options().(*descriptorpb.MessageOptions).GetMapEntry() {
				continue
			}
			typeName := message.GoIdent.GoName
			p.P(`// MarshalJSON is a custom marshaler for `, typeName)
			p.P(`func (this *`, typeName, `) MarshalJSON() ([]byte, error) {`)
			p.P(`str, err := `, marshalerName, `.MarshalToString(this)`)
			p.P(`return []byte(str), err`)
			p.P(`}`)
			// Generate UnmarshalJSON() method for this type
			p.P(`// UnmarshalJSON is a custom unmarshaler for `, typeName)
			p.P(`func (this *`, typeName, `) UnmarshalJSON(b []byte) error {`)
			p.P(`return `, unmarshalerName, `.Unmarshal(`, protogen.GoIdent{"NewReader", "bytes"}, `(b), this)`)
			p.P(`}`)
			process(message.Messages)
		}
	}
	process(file.Messages)

	// write out globals
	p.P(`var (`)
	p.P(marshalerName, ` = &`, protogen.GoIdent{"Marshaler", "github.com/golang/protobuf/jsonpb"}, `{}`)
	p.P(unmarshalerName, ` = &`, protogen.GoIdent{"Unmarshaler", "github.com/golang/protobuf/jsonpb"}, `{AllowUnknownFields: true}`)
	p.P(`)`)
}

func FileName(file *protogen.File) string {
	fname := path.Base(file.Proto.GetName())
	fname = strings.Replace(fname, ".proto", "", -1)
	fname = strings.Replace(fname, "-", "_", -1)
	fname = strings.Replace(fname, ".", "_", -1)
	return CamelCase(fname)
}

// CamelCase returns the CamelCased name.
// If there is an interior underscore followed by a lower case letter,
// drop the underscore and convert the letter to upper case.
// There is a remote possibility of this rewrite causing a name collision,
// but it's so remote we're prepared to pretend it's nonexistent - since the
// C++ generator lowercases names, it's extremely unlikely to have two fields
// with different capitalizations.
// In short, _my_field_name_2 becomes XMyFieldName_2.
func CamelCase(s string) string {
	if s == "" {
		return ""
	}
	t := make([]byte, 0, 32)
	i := 0
	if s[0] == '_' {
		// Need a capital letter; drop the '_'.
		t = append(t, 'X')
		i++
	}
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) && isASCIILower(s[i+1]) {
			continue // Skip the underscore in s.
		}
		if isASCIIDigit(c) {
			t = append(t, c)
			continue
		}
		// Assume we have a letter now - if not, it's a bogus identifier.
		// The next word is a sequence of characters that must start upper case.
		if isASCIILower(c) {
			c ^= ' ' // Make it a capital letter.
		}
		t = append(t, c) // Guaranteed not lower case.
		// Accept lower case sequence that follows.
		for i+1 < len(s) && isASCIILower(s[i+1]) {
			i++
			t = append(t, s[i])
		}
	}
	return string(t)
}

// Is c an ASCII lower-case letter?
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// Is c an ASCII digit?
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
