//	Unmarshal arbitrary XML docs to map[string]interface{} or JSON and extract values (using wildcards, if necessary).
// Copyright 2012-2018 Charles Banning. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file

/*
   Unmarshal dynamic / arbitrary XML docs and extract values (using wildcards, if necessary).
   THIS IS ONLY PROVIDED TO FACILIATE MIGRATING TO "mxj" PACKAGE FROM "x2j" PACKAGE.

   NOTICE: 03mar18, package mostly replicates github.com/clbanning/x2j using github.com/karthick18/mxj
                    (Note: there is no concept of Node or Tree; only direct decoding to map[string]interface{}.)

   One useful function is:

       - Unmarshal(doc []byte, v interface{}) error  
         where v is a pointer to a variable of type 'map[string]interface{}', 'string', or
         any other type supported by xml.Unmarshal().

   To retrieve a value for specific tag use: 

       - DocValue(doc, path string, attrs ...string) (interface{},error) 
       - MapValue(m map[string]interface{}, path string, attr map[string]interface{}, recast ...bool) (interface{}, error)

   The 'path' argument is a period-separated tag hierarchy - also known as dot-notation.
   It is the program's responsibility to cast the returned value to the proper type; possible 
   types are the normal JSON unmarshaling types: string, float64, bool, []interface, map[string]interface{}.  

   To retrieve all values associated with a tag occurring anywhere in the XML document use:

       - ValuesForTag(doc, tag string) ([]interface{}, error)
       - ValuesForKey(m map[string]interface{}, key string) []interface{}

       Demos: http://play.golang.org/p/m8zP-cpk0O
              http://play.golang.org/p/cIteTS1iSg
              http://play.golang.org/p/vd8pMiI21b

   Returned values should be one of map[string]interface, []interface{}, or string.

   All the values assocated with a tag-path that may include one or more wildcard characters - 
   '*' - can also be retrieved using:

       - ValuesFromTagPath(doc, path string, getAttrs ...bool) ([]interface{}, error)
       - ValuesFromKeyPath(map[string]interface{}, path string, getAttrs ...bool) []interface{}

       Demos: http://play.golang.org/p/kUQnZ8VuhS
   	        http://play.golang.org/p/l1aMHYtz7G

   NOTE: care should be taken when using "*" at the end of a path - i.e., "books.book.*".  See
   the x2jpath_test.go case on how the wildcard returns all key values and collapses list values;
   the same message structure can load a []interface{} or a map[string]interface{} (or an interface{}) 
   value for a tag.

   See the test cases in "x2jpath_test.go" and programs in "example" subdirectory for more.

   XML PARSING CONVENTIONS

      - Attributes are parsed to map[string]interface{} values by prefixing a hyphen, '-',
        to the attribute label.
      - If the element is a simple element and has attributes, the element value
        is given the key '#text' for its map[string]interface{} representation.  (See
        the 'atomFeedString.xml' test data, below.)

    io.Reader HANDLING

    ToMap(), ToJson(), and ToJsonIndent() provide parsing of messages from an io.Reader.
    If you want to handle a message stream, look at XmlMsgsFromReader().

    NON-UTF8 CHARACTER SETS

    Use the X2jCharsetReader variable to assign io.Reader for alternative character sets.

*/
package x2j

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/karthick18/mxj"
)

// If X2jCharsetReader != nil, it will be used to decode the doc or stream if required
//   import charset "code.google.com/p/go-charset/charset"
//   ...
//   x2j.X2jCharsetReader = charset.NewReader
//   s, err := x2j.DocToJson(doc)
var X2jCharsetReader func(charset string, input io.Reader)(io.Reader, error)

// DocToJson - return an XML doc as a JSON string.
//	If the optional argument 'recast' is 'true', then values will be converted to boolean or float64 if possible.
func DocToJson(doc string, recast ...bool) (string, error) {
	var r bool
	if len(recast) == 1 {
		r = recast[0]
	}
	m, merr := mxj.NewMapXml([]byte(doc), r)
	if m == nil || merr != nil {
		return "", merr
	}

	b, berr := m.Json()
	if berr != nil {
		return "", berr
	}

	// NOTE: don't have to worry about safe JSON marshaling with json.Marshal, since '<' and '>" are reservedin XML.
	return string(b), nil
}

// DocToJsonIndent - return an XML doc as a prettified JSON string.
//	If the optional argument 'recast' is 'true', then values will be converted to boolean or float64 if possible.
//	Note: recasting is only applied to element values, not attribute values.
func DocToJsonIndent(doc string, recast ...bool) (string, error) {
	var r bool
	if len(recast) == 1 {
		r = recast[0]
	}
	m, merr := mxj.NewMapXml([]byte(doc), r)
	if m == nil || merr != nil {
		return "", merr
	}

	b, berr := m.JsonIndent("", "  ")
	if berr != nil {
		return "", berr
	}

	// NOTE: don't have to worry about safe JSON marshaling with json.Marshal, since '<' and '>" are reservedin XML.
	return string(b), nil
}

// DocToMap - convert an XML doc into a map[string]interface{}.
// (This is analogous to unmarshalling a JSON string to map[string]interface{} using json.Unmarshal().)
//	If the optional argument 'recast' is 'true', then values will be converted to boolean or float64 if possible.
//	Note: recasting is only applied to element values, not attribute values.
func DocToMap(doc string, recast ...bool) (map[string]interface{}, error) {
	var r bool
	if len(recast) == 1 {
		r = recast[0]
	}
	return mxj.NewMapXml([]byte(doc), r)
}

// WriteMap - dumps the map[string]interface{} for examination.
//	'offset' is initial indentation count; typically: WriteMap(m).
//	NOTE: with XML all element types are 'string'.
//	But code written as generic for use with maps[string]interface{} values from json.Unmarshal().
//	Or it can handle a DocToMap(doc,true) result where values have been recast'd.
func WriteMap(m interface{}, offset ...int) string {
	var indent int
	if len(offset) == 1 {
		indent = offset[0]
	}

	var s string
	switch m.(type) {
	case nil:
		return "[nil] nil"
	case string:
		return "[string] " + m.(string)
	case float64:
		return "[float64] " + strconv.FormatFloat(m.(float64), 'e', 2, 64)
	case bool:
		return "[bool] " + strconv.FormatBool(m.(bool))
	case []interface{}:
		s += "[[]interface{}]"
		for i, v := range m.([]interface{}) {
			s += "\n"
			for i := 0; i < indent; i++ {
				s += "  "
			}
			s += "[item: " + strconv.FormatInt(int64(i), 10) + "]"
			switch v.(type) {
			case string, float64, bool:
				s += "\n"
			default:
				// noop
			}
			for i := 0; i < indent; i++ {
				s += "  "
			}
			s += WriteMap(v, indent+1)
		}
	case map[string]interface{}:
		for k, v := range m.(map[string]interface{}) {
			s += "\n"
			for i := 0; i < indent; i++ {
				s += "  "
			}
			// s += "[map[string]interface{}] "+k+" :"+WriteMap(v,indent+1)
			s += k + " :" + WriteMap(v, indent+1)
		}
	default:
		// shouldn't ever be here ...
		s += fmt.Sprintf("unknown type for: %v", m)
	}
	return s
}

// ------------------------  value extraction from XML doc --------------------------

// DocValue - return a value for a specific tag
//	'doc' is a valid XML message.
//	'path' is a hierarchy of XML tags, e.g., "doc.name".
//	'attrs' is an OPTIONAL list of "name:value" pairs for attributes.
//	Note: 'recast' is not enabled here. Use DocToMap(), NewAttributeMap(), and MapValue() calls for that.
func DocValue(doc, path string, attrs ...string) (interface{}, error) {
	m, err := mxj.NewMapXml([]byte(doc), false)
	if err != nil {
		return nil, err
	}

	a, err := NewAttributeMap(attrs...)
	if err != nil {
		return nil, err
	}
	v, verr := MapValue(m, path, a)
	if verr != nil {
		return nil, verr
	}
	return v, nil
}

// MapValue - retrieves value based on walking the map, 'm'.
//	'm' is the map value of interest.
//	'path' is a period-separated hierarchy of keys in the map.
//	'attr' is a map of attribute "name:value" pairs from NewAttributeMap().  May be 'nil'.
//	If the path can't be traversed, an error is returned.
//	Note: the optional argument 'r' can be used to coerce attribute values, 'attr', if done so for 'm'.
func MapValue(m map[string]interface{}, path string, attr map[string]interface{}, r ...bool) (interface{}, error) {
	// attribute values may have been recasted during map construction; default is 'false'.
	if len(r) == 1 && r[0] == true {
		for k, v := range attr {
			attr[k] = recast(v.(string), true)
		}
	}

	// parse the path
	keys := strings.Split(path, ".")

	// initialize return value to 'm' so a path of "" will work correctly
	var v interface{} = m
	var ok bool
	var okey string
	var isMap bool = true
	if keys[0] == "" && len(attr) == 0 {
		return v, nil
	}
	for _, key := range keys {
		if !isMap {
			return nil, errors.New("no keys beyond: " + okey)
		}
		if v, ok = m[key]; !ok {
			return nil, errors.New("no key in map: " + key)
		} else {
			switch v.(type) {
			case map[string]interface{}:
				m = v.(map[string]interface{})
				isMap = true
			default:
				isMap = false
			}
		}
		// save 'key' for error reporting
		okey = key
	}

	// match attributes; value is "#text" or nil
	if attr == nil {
		return v, nil
	}
	return hasAttributes(v, attr)
}

// recast - try to cast string values to bool or float64
func recast(s string, r bool) interface{} {
	if r {
		// handle numeric strings ahead of boolean
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return interface{}(f)
		}
		// ParseBool treats "1"==true & "0"==false
		if b, err := strconv.ParseBool(s); err == nil {
			return interface{}(b)
		}
	}
	return interface{}(s)
}

// hasAttributes() - interface{} equality works for string, float64, bool
func hasAttributes(v interface{}, a map[string]interface{}) (interface{}, error) {
	switch v.(type) {
	case []interface{}:
		// run through all entries looking one with matching attributes
		for _, vv := range v.([]interface{}) {
			if vvv, vvverr := hasAttributes(vv, a); vvverr == nil {
				return vvv, nil
			}
		}
		return nil, errors.New("no list member with matching attributes")
	case map[string]interface{}:
		// do all attribute name:value pairs match?
		nv := v.(map[string]interface{})
		for key, val := range a {
			if vv, ok := nv[key]; !ok {
				return nil, errors.New("no attribute with name: " + key[1:])
			} else if val != vv {
				return nil, errors.New("no attribute key:value pair: " + fmt.Sprintf("%s:%v", key[1:], val))
			}
		}
		// they all match; so return value associated with "#text" key.
		if vv, ok := nv["#text"]; ok {
			return vv, nil
		} else {
			// this happens when another element is value of tag rather than just a string value
			return nv, nil
		}
	}
	return nil, errors.New("no match for attributes")
}

// NewAttributeMap() - generate map of attributes=value entries as map["-"+string]string.
//	'kv' arguments are "name:value" pairs that appear as attributes, name="value".
//	If len(kv) == 0, the return is (nil, nil).
func NewAttributeMap(kv ...string) (map[string]interface{}, error) {
	if len(kv) == 0 {
		return nil, nil
	}
	m := make(map[string]interface{}, 0)
	for _, v := range kv {
		vv := strings.Split(v, ":")
		if len(vv) != 2 {
			return nil, errors.New("attribute not \"name:value\" pair: " + v)
		}
		// attributes are stored as keys prepended with hyphen
		m["-"+vv[0]] = interface{}(vv[1])
	}
	return m, nil
}

//------------------------- get values for key ----------------------------

// ValuesForTag - return all values in doc associated with 'tag'.
//	Returns nil if the 'tag' does not occur in the doc.
//	If there is an error encounted while parsing doc, that is returned.
//	If you want values 'recast' use DocToMap() and ValuesForKey().
func ValuesForTag(doc, tag string) ([]interface{}, error) {
	m, err := mxj.NewMapXml([]byte(doc))
	if err != nil {
		return nil, err
	}

	return ValuesForKey(m, tag), nil
}

// ValuesForKey - return all values in map associated with 'key'
//	Returns nil if the 'key' does not occur in the map
func ValuesForKey(m map[string]interface{}, key string) []interface{} {
	ret := make([]interface{}, 0)

	hasKey(m, key, &ret)
	if len(ret) > 0 {
		return ret
	}
	return nil
}

// hasKey - if the map 'key' exists append it to array
//          if it doesn't do nothing except scan array and map values
func hasKey(iv interface{}, key string, ret *[]interface{}) {
	switch iv.(type) {
	case map[string]interface{}:
		vv := iv.(map[string]interface{})
		if v, ok := vv[key]; ok {
			*ret = append(*ret, v)
		}
		for _, v := range iv.(map[string]interface{}) {
			hasKey(v, key, ret)
		}
	case []interface{}:
		for _, v := range iv.([]interface{}) {
			hasKey(v, key, ret)
		}
	}
}

// ======== 2013.07.01 - x2j.Unmarshal, wraps xml.Unmarshal ==============

// Unmarshal - wraps xml.Unmarshal with handling of map[string]interface{}
// and string type variables.
//	Usage: x2j.Unmarshal(doc,&m) where m of type map[string]interface{}
//	       x2j.Unmarshal(doc,&s) where s of type string (Overrides xml.Unmarshal().)
//	       x2j.Unmarshal(doc,&struct) - passed to xml.Unmarshal()
//	       x2j.Unmarshal(doc,&slice) - passed to xml.Unmarshal()
func Unmarshal(doc []byte, v interface{}) error {
	switch v.(type) {
	case *map[string]interface{}:
		m, err := mxj.NewMapXml(doc)
		vv := *v.(*map[string]interface{})
		for k, v := range m {
			vv[k] = v
		}
		return err
	case *string:
		s, err := ByteDocToJson(doc)
		*(v.(*string)) = s
		return err
	default:
		b := bytes.NewBuffer(doc)
		p := xml.NewDecoder(b)
		p.CharsetReader = X2jCharsetReader
		return p.Decode(v)
		// return xml.Unmarshal(doc, v)
	}
	return nil
}

// ByteDocToJson - return an XML doc as a JSON string.
//	If the optional argument 'recast' is 'true', then values will be converted to boolean or float64 if possible.
func ByteDocToJson(doc []byte, recast ...bool) (string, error) {
	var r bool
	if len(recast) == 1 {
		r = recast[0]
	}
	m, merr := mxj.NewMapXml(doc, r)
	if m == nil || merr != nil {
		return "", merr
	}

	b, berr := m.Json()
	if berr != nil {
		return "", berr
	}

	// NOTE: don't have to worry about safe JSON marshaling with json.Marshal, since '<' and '>" are reservedin XML.
	return string(b), nil
}

// ByteDocToMap - convert an XML doc into a map[string]interface{}.
// (This is analogous to unmarshalling a JSON string to map[string]interface{} using json.Unmarshal().)
//	If the optional argument 'recast' is 'true', then values will be converted to boolean or float64 if possible.
//	Note: recasting is only applied to element values, not attribute values.
func ByteDocToMap(doc []byte, recast ...bool) (map[string]interface{}, error) {
	var r bool
	if len(recast) == 1 {
		r = recast[0]
	}
	return mxj.NewMapXml(doc, r)
}

