package parser

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/location"
	"github.com/graphql-go/graphql/language/printer"
	"github.com/graphql-go/graphql/language/source"
)

func TestParser_BadToken(t *testing.T) {
	_, err := Parse(ParseParams{
		Source: &source.Source{
			Body: []byte("query _ {\n  me {\n    id`\n  }\n}"),
			Name: "GraphQL",
		},
	})
	if err == nil {
		t.Fatal("expected a parse error")
	}
}

func TestParser_AcceptsOptionToNotIncludeSource(t *testing.T) {
	opts := ParseOptions{
		NoSource: true,
	}
	params := ParseParams{
		Source:  "{ field }",
		Options: opts,
	}
	document, err := Parse(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oDef := ast.OperationDefinition{
		Kind: "OperationDefinition",
		Loc: &ast.Location{
			Start: 0, End: 9,
		},
		Operation:  "query",
		Directives: []*ast.Directive{},
		SelectionSet: &ast.SelectionSet{
			Kind: "SelectionSet",
			Loc: &ast.Location{
				Start: 0, End: 9,
			},
			Selections: []ast.Selection{
				&ast.Field{
					Kind: "Field",
					Loc: &ast.Location{
						Start: 2, End: 7,
					},
					Name: &ast.Name{
						Kind: "Name",
						Loc: &ast.Location{
							Start: 2, End: 7,
						},
						Value: "field",
					},
					Arguments:  []*ast.Argument{},
					Directives: []*ast.Directive{},
				},
			},
		},
	}
	expectedDocument := ast.NewDocument(&ast.Document{
		Loc: &ast.Location{
			Start: 0, End: 9,
		},
		Definitions: []ast.Node{&oDef},
	})
	if !reflect.DeepEqual(document, expectedDocument) {
		t.Fatalf("unexpected document, expected: %v, got: %v", expectedDocument, document)
	}
}

func TestParser_ParseProvidesUsefulErrors(t *testing.T) {
	opts := ParseOptions{
		NoSource: true,
	}
	params := ParseParams{
		Source:  "{",
		Options: opts,
	}
	_, err := Parse(params)

	expectedError := &gqlerrors.Error{
		Message: `Syntax Error GraphQL (1:2) Expected Name, found EOF

1: {
    ^
`,
		Positions: []int{1},
		Locations: []location.SourceLocation{{Line: 1, Column: 2}},
	}
	checkError(t, err, expectedError)

	testErrorMessagesTable := []errorMessageTest{
		{
			`{ ...MissingOn }
fragment MissingOn Type
`,
			`Syntax Error GraphQL (2:20) Expected "on", found Name "Type"`,
			false,
		},
		{
			`{ field: {} }`,
			`Syntax Error GraphQL (1:10) Expected Name, found {`,
			false,
		},
		{
			`notanoperation Foo { field }`,
			`Syntax Error GraphQL (1:1) Unexpected Name "notanoperation"`,
			false,
		},
		{
			"...",
			`Syntax Error GraphQL (1:1) Unexpected ...`,
			false,
		},
	}
	for _, test := range testErrorMessagesTable {
		if test.skipped != false {
			t.Skipf("Skipped test: %v", test.source)
		}
		_, err := Parse(ParseParams{Source: test.source})
		checkErrorMessage(t, err, test.expectedMessage)
	}

}

func TestParser_ParseProvidesUsefulErrorsWhenUsingSource(t *testing.T) {
	test := errorMessageTest{
		source.NewSource(&source.Source{
			Body: []byte("query"),
			Name: "MyQuery.graphql",
		}),
		`Syntax Error MyQuery.graphql (1:6) Expected {, found EOF`,
		false,
	}
	testErrorMessage(t, test)
}

func TestParser_ParsesVariableInlineValues(t *testing.T) {
	source := `{ field(complex: { a: { b: [ $var ] } }) }`
	// should not return error
	_, err := Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_ParsesConstantDefaultValues(t *testing.T) {
	test := errorMessageTest{
		`query Foo($x: Complex = { a: { b: [ $var ] } }) { field }`,
		`Syntax Error GraphQL (1:37) Unexpected $`,
		false,
	}
	testErrorMessage(t, test)
}

func TestParser_DoesNotAcceptFragmentsNameOn(t *testing.T) {
	test := errorMessageTest{
		`fragment on on on { on }`,
		`Syntax Error GraphQL (1:10) Unexpected Name "on"`,
		false,
	}
	testErrorMessage(t, test)
}

func TestParser_DoesNotAcceptFragmentsSpreadOfOn(t *testing.T) {
	test := errorMessageTest{
		`{ ...on }'`,
		`Syntax Error GraphQL (1:9) Expected Name, found }`,
		false,
	}
	testErrorMessage(t, test)
}

func TestParser_DoesNotAllowNullAsValue(t *testing.T) {
	test := errorMessageTest{
		`{ fieldWithNullableStringInput(input: null) }'`,
		`Syntax Error GraphQL (1:39) Unexpected Name "null"`,
		false,
	}
	testErrorMessage(t, test)
}

func TestParser_ParsesMultiByteCharacters_Unicode(t *testing.T) {

	doc := `
        # This comment has a \u0A0A multi-byte character.
        { field(arg: "Has a \u0A0A multi-byte character.") }
	`
	astDoc := parse(t, doc)

	expectedASTDoc := ast.NewDocument(&ast.Document{
		Loc: ast.NewLocation(&ast.Location{
			Start: 67,
			End:   121,
		}),
		Definitions: []ast.Node{
			ast.NewOperationDefinition(&ast.OperationDefinition{
				Loc: ast.NewLocation(&ast.Location{
					Start: 67,
					End:   119,
				}),
				Operation: "query",
				SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
					Loc: ast.NewLocation(&ast.Location{
						Start: 67,
						End:   119,
					}),
					Selections: []ast.Selection{
						ast.NewField(&ast.Field{
							Loc: ast.NewLocation(&ast.Location{
								Start: 67,
								End:   117,
							}),
							Name: ast.NewName(&ast.Name{
								Loc: ast.NewLocation(&ast.Location{
									Start: 69,
									End:   74,
								}),
								Value: "field",
							}),
							Arguments: []*ast.Argument{
								ast.NewArgument(&ast.Argument{
									Loc: ast.NewLocation(&ast.Location{
										Start: 75,
										End:   116,
									}),
									Name: ast.NewName(&ast.Name{

										Loc: ast.NewLocation(&ast.Location{
											Start: 75,
											End:   78,
										}),
										Value: "arg",
									}),
									Value: ast.NewStringValue(&ast.StringValue{

										Loc: ast.NewLocation(&ast.Location{
											Start: 80,
											End:   116,
										}),
										Value: "Has a \u0A0A multi-byte character.",
									}),
								}),
							},
						}),
					},
				}),
			}),
		},
	})

	astDocQuery := printer.Print(astDoc)
	expectedASTDocQuery := printer.Print(expectedASTDoc)

	if !reflect.DeepEqual(astDocQuery, expectedASTDocQuery) {
		t.Fatalf("unexpected document, expected: %v, got: %v", astDocQuery, expectedASTDocQuery)
	}
}

func TestParser_ParsesMultiByteCharacters_UnicodeText(t *testing.T) {

	doc := `
        # This comment has a фы世界 multi-byte character.
        { field(arg: "Has a фы世界 multi-byte character.") }
	`
	astDoc := parse(t, doc)

	expectedASTDoc := ast.NewDocument(&ast.Document{
		Loc: ast.NewLocation(&ast.Location{
			Start: 67,
			End:   121,
		}),
		Definitions: []ast.Node{
			ast.NewOperationDefinition(&ast.OperationDefinition{
				Loc: ast.NewLocation(&ast.Location{
					Start: 67,
					End:   119,
				}),
				Operation: "query",
				SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
					Loc: ast.NewLocation(&ast.Location{
						Start: 67,
						End:   119,
					}),
					Selections: []ast.Selection{
						ast.NewField(&ast.Field{
							Loc: ast.NewLocation(&ast.Location{
								Start: 67,
								End:   117,
							}),
							Name: ast.NewName(&ast.Name{
								Loc: ast.NewLocation(&ast.Location{
									Start: 69,
									End:   74,
								}),
								Value: "field",
							}),
							Arguments: []*ast.Argument{
								ast.NewArgument(&ast.Argument{
									Loc: ast.NewLocation(&ast.Location{
										Start: 75,
										End:   116,
									}),
									Name: ast.NewName(&ast.Name{

										Loc: ast.NewLocation(&ast.Location{
											Start: 75,
											End:   78,
										}),
										Value: "arg",
									}),
									Value: ast.NewStringValue(&ast.StringValue{

										Loc: ast.NewLocation(&ast.Location{
											Start: 80,
											End:   116,
										}),
										Value: "Has a фы世界 multi-byte character.",
									}),
								}),
							},
						}),
					},
				}),
			}),
		},
	})

	astDocQuery := printer.Print(astDoc)
	expectedASTDocQuery := printer.Print(expectedASTDoc)

	if !reflect.DeepEqual(astDocQuery, expectedASTDocQuery) {
		t.Fatalf("unexpected document, expected: %v, got: %v", astDocQuery, expectedASTDocQuery)
	}
}

func TestParser_ParsesKitchenSink(t *testing.T) {
	b, err := ioutil.ReadFile("../../kitchen-sink.graphql")
	if err != nil {
		t.Fatalf("unable to load kitchen-sink.graphql")
	}
	source := string(b)
	_, err = Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_AllowsNonKeywordsAnywhereNameIsAllowed(t *testing.T) {
	nonKeywords := []string{
		"on",
		"fragment",
		"query",
		"mutation",
		"subscription",
		"true",
		"false",
	}
	for _, keyword := range nonKeywords {
		fragmentName := keyword
		// You can't define or reference a fragment named `on`.
		if keyword == "on" {
			fragmentName = "a"
		}
		source := fmt.Sprintf(`query %v {
			... %v
			... on %v { field }
		}
		fragment %v on Type {
		%v(%v: $%v) @%v(%v: $%v)
		}
		`, keyword, fragmentName, keyword, fragmentName, keyword, keyword, keyword, keyword, keyword, keyword)
		_, err := Parse(ParseParams{Source: source})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParser_ParsesExperimentalSubscriptionFeature(t *testing.T) {
	source := `
      subscription Foo {
        subscriptionField
      }
    `
	_, err := Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_ParsesAnonymousMutationOperations(t *testing.T) {
	source := `
      mutation {
        mutationField
      }
    `
	_, err := Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_ParsesAnonymousSubscriptionOperations(t *testing.T) {
	source := `
      subscription {
        subscriptionField
      }
    `
	_, err := Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_ParsesNamedMutationOperations(t *testing.T) {
	source := `
      mutation Foo {
        mutationField
      }
    `
	_, err := Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_ParsesNamedSubscriptionOperations(t *testing.T) {
	source := `
      subscription Foo {
        subscriptionField
      }
    `
	_, err := Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_ParsesFieldDefinitionWithDescription(t *testing.T) {
	source := `
		type Foo implements Bar {
			"""
			foo is quite the field.
			"""
			foo: String!
		}
	`
	_, err := Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_ParsesInputValueDefinitionWithDescription(t *testing.T) {
	source := `
		type Foo implements Bar {
			foo(
				"""
				input value comment
				"""
				bar: String!
			): String!
		}
	`
	_, err := Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_ParsesEnumValueDefinitionWithDescription(t *testing.T) {
	source := `
		enum Site {
			"description 1"
			DESKTOP
			"""
			description 2
			"""
			MOBILE
		}
	`
	_, err := Parse(ParseParams{Source: source})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParser_DefinitionsWithDescriptions(t *testing.T) {
	testCases := []struct {
		name            string
		source          string
		expectedComment string
	}{
		{
			name: "directives",
			source: `
				"cool skip"
				directive @skip(if: Boolean!) on FIELD | FRAGMENT_SPREAD | INLINE_FRAGMENT
			`,
			expectedComment: "cool skip",
		},
		{
			name: "input",
			source: `
				"""
				InputType is indeed a type
				"""
				input InputType {
					key: String!
					answer: Int = 42
				}
			`,
			expectedComment: "InputType is indeed a type",
		},
		{
			name: "enum",
			source: `
				"""
				description 2
				"""
				enum Site {
					DESKTOP
					MOBILE
				}
			`,
			expectedComment: "description 2",
		},
		{
			name: "union",
			source: `
				"""
				Cruft ...
				"""
				union Cruft = Foo | Bar
			`,
			expectedComment: "Cruft ...",
		},
		{
			name: "interface",
			source: `
				"""
				Bar is a symptom of the communist agenda
				"""
				interface  Bar {
					foo: String!
				}
			`,
			expectedComment: "Bar is a symptom of the communist agenda",
		},
		{
			name: "object",
			source: `
				"""
				★ Foo ★
				"""
				type Foo implements Bar {
					foo: String!
				}
			`,
			expectedComment: "★ Foo ★",
		},
		{
			name: "scalar",
			source: `
				"""
				Returns RFC666; includes timezone offset.
				"""
				scalar TimeWithZone
			`,
			expectedComment: "Returns RFC666; includes timezone offset.",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s may have comments", tc.name), func(t *testing.T) {
			doc, err := Parse(ParseParams{Source: tc.source})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if doc == nil {
				t.Fatal("no document was returned")
			}
			for _, def := range doc.Definitions {
				fmt.Printf("%#v\n", def)
			}
			if node, ok := doc.Definitions[0].(ast.DescribableNode); !ok {
				t.Fatalf("unexpected node received %#v", doc.Definitions[0])
			} else if node.GetDescription().Value != tc.expectedComment {
				t.Fatalf(
					"parsed description '%s' does not match '%s'",
					node.GetDescription().Value,
					tc.expectedComment,
				)
			}
		})
	}
}

func TestParser_ParseCreatesAst(t *testing.T) {
	body := `{
  node(id: 4) {
    id,
    name
  }
}
`
	source := source.NewSource(&source.Source{
		Body: []byte(body),
	})
	document, err := Parse(
		ParseParams{
			Source: source,
			Options: ParseOptions{
				NoSource: true,
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oDef := ast.OperationDefinition{
		Kind: "OperationDefinition",
		Loc: &ast.Location{
			Start: 0, End: 40,
		},
		Operation:  "query",
		Directives: []*ast.Directive{},
		SelectionSet: &ast.SelectionSet{
			Kind: "SelectionSet",
			Loc: &ast.Location{
				Start: 0, End: 40,
			},
			Selections: []ast.Selection{
				&ast.Field{
					Kind: "Field",
					Loc: &ast.Location{
						Start: 4, End: 38,
					},
					Name: &ast.Name{
						Kind: "Name",
						Loc: &ast.Location{
							Start: 4, End: 8,
						},
						Value: "node",
					},
					Arguments: []*ast.Argument{
						{
							Kind: "Argument",
							Name: &ast.Name{
								Kind: "Name",
								Loc: &ast.Location{
									Start: 9, End: 11,
								},
								Value: "id",
							},
							Value: &ast.IntValue{
								Kind: "IntValue",
								Loc: &ast.Location{
									Start: 13, End: 14,
								},
								Value: "4",
							},
							Loc: &ast.Location{
								Start: 9, End: 14,
							},
						},
					},
					Directives: []*ast.Directive{},
					SelectionSet: &ast.SelectionSet{
						Kind: "SelectionSet",
						Loc: &ast.Location{
							Start: 16, End: 38,
						},
						Selections: []ast.Selection{
							&ast.Field{
								Kind: "Field",
								Loc: &ast.Location{
									Start: 22, End: 24,
								},
								Name: &ast.Name{
									Kind: "Name",
									Loc: &ast.Location{
										Start: 22, End: 24,
									},
									Value: "id",
								},
								Arguments:    []*ast.Argument{},
								Directives:   []*ast.Directive{},
								SelectionSet: nil,
							},
							&ast.Field{
								Kind: "Field",
								Loc: &ast.Location{
									Start: 30, End: 34,
								},
								Name: &ast.Name{
									Kind: "Name",
									Loc: &ast.Location{
										Start: 30, End: 34,
									},
									Value: "name",
								},
								Arguments:    []*ast.Argument{},
								Directives:   []*ast.Directive{},
								SelectionSet: nil,
							},
						},
					},
				},
			},
		},
	}
	expectedDocument := ast.NewDocument(&ast.Document{
		Loc: &ast.Location{
			Start: 0, End: 41,
		},
		Definitions: []ast.Node{&oDef},
	})
	if !reflect.DeepEqual(document, expectedDocument) {
		t.Fatalf("unexpected document, expected: %v, got: %v", expectedDocument, document.Definitions)
	}

}

func TestParser_DoesNotAcceptStringAsDefinition(t *testing.T) {
	test := errorMessageTest{
		`String`,
		`Syntax Error GraphQL (1:1) Unexpected Name "String"`,
		false,
	}
	testErrorMessage(t, test)
}

type errorMessageTest struct {
	source          interface{}
	expectedMessage string
	skipped         bool
}

func testErrorMessage(t *testing.T, test errorMessageTest) {
	if test.skipped != false {
		t.Skipf("Skipped test: %v", test.source)
	}
	_, err := Parse(ParseParams{Source: test.source})
	checkErrorMessage(t, err, test.expectedMessage)
}

func checkError(t *testing.T, err error, expectedError *gqlerrors.Error) {
	if expectedError == nil {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return // ok
	}
	// else expectedError != nil
	if err == nil {
		t.Fatalf("unexpected nil error\nexpected:\n%v\n\ngot:\n%v", expectedError, err)
	}
	if err.Error() != expectedError.Message {
		t.Fatalf("unexpected error.\nexpected:\n%v\n\ngot:\n%v", expectedError, err.Error())
	}
	gErr := toError(err)
	if gErr == nil {
		t.Fatalf("unexpected nil Error")
	}
	if len(expectedError.Positions) > 0 && !reflect.DeepEqual(gErr.Positions, expectedError.Positions) {
		t.Fatalf("unexpected Error.Positions.\nexpected:\n%v\n\ngot:\n%v", expectedError.Positions, gErr.Positions)
	}
	if len(expectedError.Locations) > 0 && !reflect.DeepEqual(gErr.Locations, expectedError.Locations) {
		t.Fatalf("unexpected Error.Locations.\nexpected:\n%v\n\ngot:\n%v", expectedError.Locations, gErr.Locations)
	}
}

func checkErrorMessage(t *testing.T, err error, expectedMessage string) {
	if err == nil {
		t.Fatalf("unexpected nil error\nexpected:\n%v\n\ngot:\n%v", expectedMessage, err)
	}
	if err.Error() != expectedMessage {
		// only check first line of error message
		lines := strings.Split(err.Error(), "\n")
		if lines[0] != expectedMessage {
			t.Fatalf("unexpected error.\nexpected:\n%v\n\ngot:\n%v", expectedMessage, lines[0])
		}
	}
}

func toError(err error) *gqlerrors.Error {
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *gqlerrors.Error:
		return err
	default:
		return nil
	}
}
