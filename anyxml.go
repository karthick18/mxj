package mxj

import (
	"bytes"
	"encoding/xml"
	"reflect"
)

const (
	DefaultElementTag = "element"
)

// Encode arbitrary value as XML.
//
// Note: unmarshaling the resultant
// XML may not return the original value, since tag labels may have been injected
// to create the XML representation of the value.
/*
 Encode an arbitrary JSON object.
	package main

	import (
		"encoding/json"
		"fmt"
		"github.com/karthick18/mxj"
	)

	func main() {
		jsondata := []byte(`[
			{ "somekey":"somevalue" },
			"string",
			3.14159265,
			true
		]`)
		var i interface{}
		err := json.Unmarshal(jsondata, &i)
		if err != nil {
			// do something
		}
		x, err := mxj.AnyXmlIndent(i, "", "  ", "mydoc")
		if err != nil {
			// do something else
		}
		fmt.Println(string(x))
	}

	output:
		<mydoc>
		  <somekey>somevalue</somekey>
		  <element>string</element>
		  <element>3.14159265</element>
		  <element>true</element>
		</mydoc>

An extreme example is available in examples/goofy_map.go.
*/
// Alternative values for DefaultRootTag and DefaultElementTag can be set as:
// AnyXml( v, myRootTag, myElementTag).
func AnyXml(v interface{}, tags ...string) ([]byte, error) {
	var rt, et string
	if len(tags) == 1 || len(tags) == 2 {
		rt = tags[0]
	} else {
		rt = DefaultRootTag
	}
	if len(tags) == 2 {
		et = tags[1]
	} else {
		et = DefaultElementTag
	}

	if v == nil {
		if useGoXmlEmptyElemSyntax {
			return []byte("<" + rt + "></" + rt + ">"), nil
		}
		return []byte("<" + rt + "/>"), nil
	}
	if reflect.TypeOf(v).Kind() == reflect.Struct {
		return xml.Marshal(v)
	}

	var err error
	s := new(bytes.Buffer)
	p := new(pretty)

	var b []byte
	switch v.(type) {
	case []interface{}:
		if _, err = s.WriteString("<" + rt + ">"); err != nil {
			return nil, err
		}
		for _, vv := range v.([]interface{}) {
			switch vv.(type) {
			case map[string]interface{}:
				m := vv.(map[string]interface{})
				if len(m) == 1 {
					for tag, val := range m {
						err = marshalMapToXmlIndent(false, s, tag, val, p)
					}
				} else {
					err = marshalMapToXmlIndent(false, s, et, vv, p)
				}
			default:
				err = marshalMapToXmlIndent(false, s, et, vv, p)
			}
			if err != nil {
				break
			}
		}
		if _, err = s.WriteString("</" + rt + ">"); err != nil {
			return nil, err
		}
		b = s.Bytes()
	case map[string]interface{}:
		m := Map(v.(map[string]interface{}))
		b, err = m.Xml(rt)
	default:
		err = marshalMapToXmlIndent(false, s, rt, v, p)
		b = s.Bytes()
	}

	return b, err
}

// Encode an arbitrary value as a pretty XML string.
// Alternative values for DefaultRootTag and DefaultElementTag can be set as:
// AnyXmlIndent( v, "", "  ", myRootTag, myElementTag).
func AnyXmlIndent(v interface{}, prefix, indent string, tags ...string) ([]byte, error) {
	var rt, et string
	if len(tags) == 1 || len(tags) == 2 {
		rt = tags[0]
	} else {
		rt = DefaultRootTag
	}
	if len(tags) == 2 {
		et = tags[1]
	} else {
		et = DefaultElementTag
	}

	if v == nil {
		if useGoXmlEmptyElemSyntax {
			return []byte(prefix + "<" + rt + "></" + rt + ">"), nil
		}
		return []byte(prefix + "<" + rt + "/>"), nil
	}
	if reflect.TypeOf(v).Kind() == reflect.Struct {
		return xml.MarshalIndent(v, prefix, indent)
	}

	var err error
	s := new(bytes.Buffer)
	p := new(pretty)
	p.indent = indent
	p.padding = prefix

	var b []byte
	switch v.(type) {
	case []interface{}:
		if _, err = s.WriteString("<" + rt + ">\n"); err != nil {
			return nil, err
		}
		p.Indent()
		for _, vv := range v.([]interface{}) {
			switch vv.(type) {
			case map[string]interface{}:
				m := vv.(map[string]interface{})
				if len(m) == 1 {
					for tag, val := range m {
						err = marshalMapToXmlIndent(true, s, tag, val, p)
					}
				} else {
					p.start = 1 // we 1 tag in
					err = marshalMapToXmlIndent(true, s, et, vv, p)
					// *s += "\n"
					if _, err = s.WriteString("\n"); err != nil {
						return nil, err
					}
				}
			default:
				p.start = 0 // in case trailing p.start = 1
				err = marshalMapToXmlIndent(true, s, et, vv, p)
			}
			if err != nil {
				break
			}
		}
		if _, err = s.WriteString(`</` + rt + `>`); err != nil {
			return nil, err
		}
		b = s.Bytes()
	case map[string]interface{}:
		m := Map(v.(map[string]interface{}))
		b, err = m.XmlIndent(prefix, indent, rt)
	default:
		err = marshalMapToXmlIndent(true, s, rt, v, p)
		b = s.Bytes()
	}

	return b, err
}
