// Copyright (c) 2017 Jimmy Zelinskie
// Copyright (c) 2015 Ash Berlin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package jq

/*
#cgo LDFLAGS: -ljq -lonig

#include <stdlib.h>

#include <jv.h>
#include <jq.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

// Helper functions for dealing with JV objects. You can't use this from
// another go package as the cgo types are 'unique' per go package

// JvKind represents the type of value that a `Jv` contains.
type JvKind int

// Jv represents a JSON value from libjq.
//
// The go wrapper uses the same memory management semantics as the underlying C
// library, so you should familiarize yourself with
// https://github.com/stedolan/jq/wiki/C-API:-jv#memory-management. In summary
// this package and all JQ functions operate on the assumption that any jv value
// you pass to a function is then owned by that function -- if you do not wish
// this to be the case call Copy() on it first.
type Jv struct {
	jv C.jv
}

const (
	// JvKindInvalid is returned when you've tried something that does not make
	// make sense (e.g. calling jv_array_get with an out of bounds index).
	JvKindInvalid JvKind = C.JV_KIND_INVALID

	// JvKindNull represents the JSON value "null".
	JvKindNull JvKind = C.JV_KIND_NULL

	// JvKindFalse represents the JSON value "false".
	JvKindFalse JvKind = C.JV_KIND_FALSE

	// JvKindTrue represents the JSON value "true".
	JvKindTrue JvKind = C.JV_KIND_TRUE

	// JvKindNumber represents the JSON type "number".
	JvKindNumber JvKind = C.JV_KIND_NUMBER

	// JvKindString represents the JSON type "string".
	JvKindString JvKind = C.JV_KIND_STRING

	// JvKindArray represents the JSON type "array".
	JvKindArray JvKind = C.JV_KIND_ARRAY

	// JvKindObject represents the JSON type "object".
	JvKindObject JvKind = C.JV_KIND_OBJECT
)

// String returns a string representation of what type this Jv contains
func (kind JvKind) String() string {
	// Rather than rely on converting from a C string to go every time, store our
	// own list
	switch kind {
	case JvKindInvalid:
		return "<invalid>"
	case JvKindNull:
		return "null"
	case JvKindFalse:
		return "boolean"
	case JvKindTrue:
		return "boolean"
	case JvKindNumber:
		return "number"
	case JvKindString:
		return "string"
	case JvKindArray:
		return "array"
	case JvKindObject:
		return "object"
	default:
		return "<unkown>"
	}
}

// JvNull returns a value representing a JSON null
func JvNull() *Jv {
	return &Jv{C.jv_null()}
}

// JvInvalid returns an invalid jv object without an error property
func JvInvalid() *Jv {
	return &Jv{C.jv_invalid()}
}

// JvInvalidWithMessage creates an "invalid" jv with the given error message.
//
// msg can be a string or an object
//
// Consumes `msg`
func JvInvalidWithMessage(msg *Jv) *Jv {
	return &Jv{C.jv_invalid_with_msg(msg.jv)}
}

// JvFromString returns a new jv string-typed value containing the given go
// string.
func JvFromString(str string) *Jv {
	cs := C.CString(str)
	defer C.free(unsafe.Pointer(cs))
	return &Jv{C.jv_string_sized(cs, C.int(len(str)))}
}

// JvFromFloat returns a new jv number-typed value containing the given float
// value.
func JvFromFloat(n float64) *Jv {
	return &Jv{C.jv_number(C.double(n))}
}

// JvFromBool returns a new jv of "true" or "false" kind depending on the given
// boolean value
func JvFromBool(b bool) *Jv {
	if b {
		return &Jv{C.jv_true()}
	}
	return &Jv{C.jv_false()}
}

func jvFromArray(val reflect.Value) (*Jv, error) {
	len := val.Len()
	ret := &Jv{C.jv_array_sized(C.int(len))}
	for i := 0; i < len; i++ {
		newjv, err := JvFromInterface(
			val.Index(i).Interface(),
		)
		if err != nil {
			// TODO: error context
			ret.Free()
			return nil, err
		}
		ret = &Jv{C.jv_array_set(ret.jv, C.int(i), newjv.jv)}
	}
	return ret, nil
}

func jvFromMap(val reflect.Value) (*Jv, error) {
	keys := val.MapKeys()
	ret := JvObject()

	for _, key := range keys {
		keyjv := JvFromString(key.String())
		valjv, err := JvFromInterface(val.MapIndex(key).Interface())
		if err != nil {
			// TODO: error context
			keyjv.Free()
			ret.Free()
			return nil, err
		}
		ret = ret.ObjectSet(keyjv, valjv)
	}

	return ret, nil
}

// JvFromInterface uses reflection to dynamically transform an Go types into a
// Jv.
func JvFromInterface(intf interface{}) (*Jv, error) {
	if intf == nil {
		return JvNull(), nil
	}

	switch x := intf.(type) {
	case float32:
		return JvFromFloat(float64(x)), nil
	case float64:
		return JvFromFloat(x), nil
	case uint:
		return JvFromFloat(float64(x)), nil
	case int:
		return JvFromFloat(float64(x)), nil
	case int8:
		return JvFromFloat(float64(x)), nil
	case uint8:
		return JvFromFloat(float64(x)), nil
	case int16:
		return JvFromFloat(float64(x)), nil
	case uint16:
		return JvFromFloat(float64(x)), nil
	case int32:
		return JvFromFloat(float64(x)), nil
	case uint32:
		return JvFromFloat(float64(x)), nil
	case int64:
		return JvFromFloat(float64(x)), nil
	case uint64:
		return JvFromFloat(float64(x)), nil
	case string:
		return JvFromString(x), nil
	case []byte:
		return JvFromString(string(x)), nil
	case bool:
		return JvFromBool(x), nil
	}

	val := reflect.ValueOf(intf)
	switch val.Kind() {
	case reflect.Array, reflect.Slice:
		return jvFromArray(val)
	case reflect.Map:
		return jvFromMap(val)
	default:
		return nil, errors.New("JvFromInterface can't handle " + val.Kind().String())
	}
}

func _ConvertError(inv C.jv) error {
	// We might want to not call this as it prefixes things with "jq: "
	jv := &Jv{C.jq_format_error(inv)}
	defer jv.Free()

	return errors.New(jv._string())
}

// JvFromJSONString takes a JSON string and returns the jv representation of
// it.
func JvFromJSONString(str string) (*Jv, error) {
	cs := C.CString(str)
	defer C.free(unsafe.Pointer(cs))
	jv := C.jv_parse(cs)

	if C.jv_is_valid(jv) == 0 {
		return nil, _ConvertError(jv)
	}
	return &Jv{jv}, nil
}

// JvFromJSONBytes takes a utf-8 byte sequence containing JSON and returns the
// jv representation of it.
func JvFromJSONBytes(b []byte) (*Jv, error) {
	jv := C.jv_parse((*C.char)(unsafe.Pointer(&b[0])))

	if C.jv_is_valid(jv) == 0 {
		return nil, _ConvertError(jv)
	}
	return &Jv{jv}, nil
}

// Free this reference to a Jv value.
//
// Don't call this more than once per jv - might not actually free the memory
// as libjq uses reference counting. To make this more like the libjq interface
// we return a nil pointer.
func (jv *Jv) Free() *Jv {
	C.jv_free(jv.jv)
	return nil
}

// Kind returns a JvKind saying what type this jv contains.
//
// Does not consume the invocant.
func (jv *Jv) Kind() JvKind {
	return JvKind(C.jv_get_kind(jv.jv))
}

// Copy returns a *Jv so that the original won't get freed.
//
// Does not consume the invocant.
func (jv *Jv) Copy() *Jv {
	C.jv_copy(jv.jv)
	// Becasue jv uses ref counting under the hood we can return the same value
	return jv
}

// IsValid returns true if this Jv represents a valid JSON type, or false if it
// is unitiaizlied or if it represents an error type
//
// Does not consume the invocant.
func (jv *Jv) IsValid() bool {
	return C.jv_is_valid(jv.jv) != 0
}

// GetInvalidMessageAsString gets the error message for this Jv. If there is none it
// will return ("", false). Otherwise it will return the message as a string and true,
// converting non-string values if necessary. If you want the message in it's
// native Jv type use `GetInvalidMessage()`
//
// Consumes the invocant.
func (jv *Jv) GetInvalidMessageAsString() (string, bool) {
	msg := C.jv_invalid_get_msg(jv.jv)
	defer C.jv_free(msg)

	if C.jv_get_kind(msg) == C.JV_KIND_NULL {
		return "", false
	} else if C.jv_get_kind(msg) != C.JV_KIND_STRING {
		msg = C.jv_dump_string(msg, 0)
	}
	return C.GoString(C.jv_string_value(msg)), true
}

// GetInvalidMessage returns the message associcated
func (jv *Jv) GetInvalidMessage() *Jv {
	return &Jv{C.jv_invalid_get_msg(jv.jv)}
}

func (jv *Jv) _string() string {
	// Raw string value. If called on
	cs := C.jv_string_value(jv.jv)
	// Don't free cs - freed when the jv is
	return C.GoString(cs)
}

// If jv is a string, return its value. Will not stringify other types
//
// Does not consume the invocant.
func (jv *Jv) String() (string, error) {
	// Doing this might be a bad idea as it means we almost implement the Stringer
	// interface but not quite (cos the error type)

	// If we don't do this check JV will assert
	if C.jv_get_kind(jv.jv) != C.JV_KIND_STRING {
		return "", fmt.Errorf("Cannot return String for jv of type %s", jv.Kind())
	}

	return jv._string(), nil
}

// ToGoVal converts a jv into it's closest Go approximation
//
// Does not consume the invocant.
func (jv *Jv) ToGoVal() interface{} {
	switch kind := C.jv_get_kind(jv.jv); kind {
	case C.JV_KIND_NULL:
		return nil
	case C.JV_KIND_FALSE:
		return false
	case C.JV_KIND_TRUE:
		return true
	case C.JV_KIND_NUMBER:
		dbl := C.jv_number_value(jv.jv)

		if C.jv_is_integer(jv.jv) == 0 {
			return float64(dbl)
		}
		return int(dbl)
	case C.JV_KIND_STRING:
		return jv._string()
	case C.JV_KIND_ARRAY:
		len := jv.Copy().ArrayLength()
		ary := make([]interface{}, len)
		for i := 0; i < len; i++ {
			v := jv.Copy().ArrayGet(i)
			ary[i] = v.ToGoVal()
			v.Free()
		}
		return ary
	case C.JV_KIND_OBJECT:
		obj := make(map[string]interface{})
		for iter := C.jv_object_iter(jv.jv); C.jv_object_iter_valid(jv.jv, iter) != 0; iter = C.jv_object_iter_next(jv.jv, iter) {
			k := Jv{C.jv_object_iter_key(jv.jv, iter)}
			v := Jv{C.jv_object_iter_value(jv.jv, iter)}
			// jv_object_iter_key already asserts that the kind is string, so using _string is OK here
			obj[k._string()] = v.ToGoVal()
			k.Free()
			v.Free()
		}
		return obj
	default:
		panic(fmt.Sprintf("Unknown JV kind %d", kind))
	}
}

// JvPrintFlags represents the type of flags used for configuring how Jvs are
// printed.
type JvPrintFlags int

const (
	// JvPrintNone is a blank set of flags.
	JvPrintNone JvPrintFlags = 0

	// JvPrintPretty prints across multiple lines.
	JvPrintPretty JvPrintFlags = C.JV_PRINT_PRETTY

	// JvPrintASCII escapes non-ascii printable characters.
	JvPrintASCII JvPrintFlags = C.JV_PRINT_ASCII

	// JvPrintColour includes ANSI color escapes based on data types.
	JvPrintColour JvPrintFlags = C.JV_PRINT_COLOUR

	// JvPrintSorted sorts the output keys.
	JvPrintSorted JvPrintFlags = C.JV_PRINT_SORTED

	// JvPrintInvalid prints invalid values as "<invalid>".
	JvPrintInvalid JvPrintFlags = C.JV_PRINT_INVALID

	// JvPrintRefCount displays refcounts of objects in parenthesis.
	JvPrintRefCount JvPrintFlags = C.JV_PRINT_REFCOUNT

	// JvPrintTab indents with tabs.
	JvPrintTab JvPrintFlags = C.JV_PRINT_TAB

	// JvPrintIsATty TODO(jzelinskie): figure out what this even does
	JvPrintIsATty JvPrintFlags = C.JV_PRINT_ISATTY

	// JvPrintSpace0 indents with zero extra chars beyond the parent bracket.
	JvPrintSpace0 JvPrintFlags = C.JV_PRINT_SPACE0

	// JvPrintSpace1 indents with one extra chars beyond the parent bracket.
	JvPrintSpace1 JvPrintFlags = C.JV_PRINT_SPACE1

	// JvPrintSpace2 indents with two extra chars beyond the parent bracket.
	JvPrintSpace2 JvPrintFlags = C.JV_PRINT_SPACE2
)

// Dump produces a human readable version of the string with the requested formatting.
//
// Consumes the invocant
func (jv *Jv) Dump(flags JvPrintFlags) string {
	jvStr := Jv{C.jv_dump_string(jv.jv, C.int(flags))}
	defer jvStr.Free()
	return jvStr._string()
}

// JvArray creates a new, empty array-typed JV
func JvArray() *Jv {
	return &Jv{C.jv_array()}
}

// ArrayAppend appends a single value to the end of the array.
//
// If jv is not an array this will cause an assertion.
//
// Consumes the invocant
func (jv *Jv) ArrayAppend(val *Jv) *Jv {
	return &Jv{C.jv_array_append(jv.jv, val.jv)}
}

// ArrayLength returns the number of elements in the array.
//
// Consumes the invocant
func (jv *Jv) ArrayLength() int {
	return int(C.jv_array_length(jv.jv))
}

// ArrayGet returns the element at the given array index.
//
// If the index is out of bounds it will return an Invalid Jv object (with no
// error message set).
//
// `idx` cannot be negative.
//
// Consumes the invocant
func (jv *Jv) ArrayGet(idx int) *Jv {
	return &Jv{C.jv_array_get(jv.jv, C.int(idx))}
}

// JvObject allocates a new Jv of type object.
func JvObject() *Jv {
	return &Jv{C.jv_object()}
}

// ObjectSet will add val to the object under the given key.
//
// This is the equivalent of `jv[key] = val`.
//
// Consumes invocant and both key and val
func (jv *Jv) ObjectSet(key *Jv, val *Jv) *Jv {
	return &Jv{C.jv_object_set(jv.jv, key.jv, val.jv)}
}
