package writer

import "fmt"

// Dict is a PDF dictionary that remembers insertion order. Setting an
// existing key replaces its value and keeps its position; deleting a key and
// setting it again moves it to the end.
type Dict struct {
	keys []Name
	vals map[Name]Object
}

// NewDict builds a dictionary from alternating keys and values. Keys are
// [Name] or string; values are any [Object] or an int, int64, float64, bool
// or nil (written as null). It panics on malformed arguments, which are a
// bug in the caller.
func NewDict(kv ...any) *Dict {
	if len(kv)%2 != 0 {
		panic("writer: NewDict needs key/value pairs")
	}
	d := &Dict{}
	for i := 0; i < len(kv); i += 2 {
		var k Name
		switch key := kv[i].(type) {
		case Name:
			k = key
		case string:
			k = Name(key)
		default:
			panic(fmt.Sprintf("writer: NewDict key %d is %T, want Name or string", i/2, kv[i]))
		}
		d.Set(k, toObject(kv[i+1]))
	}
	return d
}

func toObject(v any) Object {
	switch v := v.(type) {
	case nil:
		return Null{}
	case Object:
		return v
	case int:
		return Int(v)
	case int64:
		return Int(v)
	case float64:
		return Real(v)
	case bool:
		return Bool(v)
	}
	panic(fmt.Sprintf("writer: unsupported value of type %T", v))
}

// Set stores v under k.
func (d *Dict) Set(k Name, v Object) {
	if d.vals == nil {
		d.vals = make(map[Name]Object)
	}
	if _, ok := d.vals[k]; !ok {
		d.keys = append(d.keys, k)
	}
	d.vals[k] = v
}

// Get returns the value stored under k.
func (d *Dict) Get(k Name) (Object, bool) {
	if d == nil {
		return nil, false
	}
	v, ok := d.vals[k]
	return v, ok
}

// Delete removes k.
func (d *Dict) Delete(k Name) {
	if d == nil {
		return
	}
	if _, ok := d.vals[k]; !ok {
		return
	}
	delete(d.vals, k)
	for i, key := range d.keys {
		if key == k {
			d.keys = append(d.keys[:i], d.keys[i+1:]...)
			break
		}
	}
}

// Keys returns the keys in insertion order.
func (d *Dict) Keys() []Name {
	if d == nil {
		return nil
	}
	return append([]Name(nil), d.keys...)
}

// Len returns the number of entries.
func (d *Dict) Len() int {
	if d == nil {
		return 0
	}
	return len(d.keys)
}

// clone returns a shallow copy; a nil receiver gives an empty dictionary.
func (d *Dict) clone() *Dict {
	c := &Dict{}
	if d == nil {
		return c
	}
	c.keys = append([]Name(nil), d.keys...)
	c.vals = make(map[Name]Object, len(d.vals))
	for k, v := range d.vals {
		c.vals[k] = v
	}
	return c
}

func (d *Dict) appendTo(dst []byte, compress bool) []byte {
	dst = append(dst, "<<"...)
	if d != nil {
		for _, k := range d.keys {
			dst = appendName(dst, string(k))
			dst = append(dst, ' ')
			dst = appendObject(dst, d.vals[k], compress)
		}
	}
	return append(dst, ">>"...)
}
