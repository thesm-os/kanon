// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package kanon generates the codecs of Go struct types for the kanon
// command. [Generate] loads the package of a source file with a kanon
// directive and returns two files for the types that [Options.Types] names:
// <base>.kanon.go declares the methods of kanon.Cloner on each struct type
// and the ValidateKanon method of kanon.Validator on each named type that is
// not a struct, and <base>.kanon_test.go runs the conformance suites of
// package kanontest on them. [Check] checks the field numbers of the same
// types against the
// numbers that the code files of a base revision record. [ParseOptions]
// parses the flags of the kanon command, which a go:generate directive
// passes in the same form.
//
// # Analysis
//
// The analysis reads the struct types from the syntax trees and the type
// information of the package, and maps each field to its encoding. A field
// is encoded when it is exported or has a kanon tag, which opts an
// unexported field in, has no `kanon:"-"` tag, and is neither a function
// nor a channel. Only the code of its package can read an unexported field,
// so the analysis rejects one that a tag opts in within a struct of another
// package that the file encodes inline or orders as a map key.
//
// A struct type that a -type flag of its package names, or whose pointer
// has the methods of a kanon codec, encodes through those methods. Any
// other struct type in the types of the fields is an inline struct: the
// code file declares functions that write the encoding its own codec would
// write. A type with the binary, gob or text methods of packages encoding
// and encoding/gob encodes itself through them, as gob encodes it.
//
// A named type that is not a struct is a kanon.Validator when a -type flag
// of its package names it, or when it has ValidateKanon on a value receiver.
// It encodes as its underlying type, ahead of its binary, gob and text
// methods, and the generated code calls ValidateKanon on each value of the
// type that it encodes or decodes. The analysis rejects ValidateKanon on a
// struct type, on a pointer receiver or with another signature, and on a
// type whose only value is its zero value. A named interface type whose
// method set has ValidateKanon encodes as any other interface.
//
// # Field numbers
//
// A kanon tag sets the number of a field. A field without one takes the
// smallest free number in declaration order. The code file records the
// numbers of every struct that it encodes in a //kanon:numbers line:
//
//   - a -type struct and an inline struct of the package under their type
//     names;
//   - an inline struct of another package under its import path and type
//     name;
//   - the instantiations of a generic struct under the key of the generic
//     struct, which numbers them alike;
//   - an anonymous struct under the path of the first field whose type
//     contains it: Order.Meta for a field, and Order.Lines[],
//     Order.Index[key] and Order.Index[value] for the elements of a slice or
//     an array and the keys and values of a map;
//   - the concrete types of the tag option types of a field under the path
//     of the field, each type under its type string, with the import path of
//     each package other than the package of the file.
//
// A field keeps its recorded number when its struct changes, and so does a
// concrete type. The number of a removed field or type goes to no other one
// unless a tag names it. [Check] compares the numbers with the code files of
// a base revision.
//
// # Generated code
//
// The code file imports the runtime packages go.thesmos.sh/kanon and
// go.thesmos.sh/kanon/wire and the standard library. It declares two
// kanon.EnforceVersion constants for version 1 of the generator. Each
// -type struct gets the exported methods of kanon.Cloner and the unexported
// methods encodeKanon, decodeKanon, mergeKanon, fieldsKanon and cloneKanon,
// which the code of the structs of the package that contain it calls. The
// helpers of the file, the functions of its inline structs, slices, arrays,
// maps, pointers and interfaces, take the name of the source file as a
// prefix, so that two code files of one package declare distinct names.
// With -views, each -type struct also gets a view type, <T>View, and its
// index type, <T>Index, which IndexKanon of the view fills from one scan.
//
// A type that encodes itself and declares SizeKanon() int, a kanon.Sizer, is
// sized with that method and appended into its room in place. The presence
// of a value of a type that declares IsZero() bool, and whose == compares
// every bit, is tested with that method in place of ==.
//
// An encode writes backward from the end of its buffer, in ascending field
// number, so that the length of a nested value is known when its prefix is
// written. A decode tracks the pointers, interfaces, structs and maps that
// its input contains in a seen bitmap, reuses their memory, and sets the
// ones that the input does not contain to their zero values at its end.
//
// # Failure semantics
//
// Every error message starts with "kanon: ", except the error of gofmt for a
// generated file that it cannot parse, which marks a defect of kanon. An
// error about a declaration of the source contains its file, line and
// column, and the name of the struct or the field. A declaration of another
// package is cited at column 1 of its line, the position that its export
// data records. An error of [Check] cites a struct or a list by the key of
// its numbers line.
//
// # Platforms
//
// The generated files do not depend on the platform that runs kanon. A
// directive names the kanon command with slashes or backslashes, with or
// without the suffix .exe, and the files end their lines with "\n", which
// the .gitattributes of a repository keeps on checkout.
//
// # Dependency position
//
// kanon imports the standard library only, among it go/types, go/parser and
// go/importer, and runs go list to find the files of a package and the
// export data of its dependencies. The kanon command imports it.
package kanon
