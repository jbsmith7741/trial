package trial

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// ContainsFn has been renamed to Contains
func ContainsFn(x, y interface{}) (bool, string) { return Contains(x, y) }

// Contains determines if y is a subset of x.
// x is a string -> y is a string that is equal to or a subset of x (string.Contains)
// x is a slice or array -> y is contained in x
// x is a map -> y is a map and is contained in x
func Contains(x, y interface{}) (bool, string) {
	// if nothing is expected we have a match
	if y == nil {
		return true, ""
	}
	r := contains(x, y)
	return r.Equal(), r.String()
}

func contains(x, y interface{}) *diff {
	d := newDiff()
	valX := reflect.ValueOf(x)
	valY := reflect.ValueOf(y)
	switch valX.Kind() {
	case reflect.String:
		s, ok := y.(string)
		if !ok {
			if v, ok := y.(fmt.Stringer); ok {
				s = v.String()
			} else {
				return d.Errorf("type mismatch -%T +%T", x, y)
			}
		}
		if strings.Contains(x.(string), s) {
			return nil
		}
		return d.Errorf(cmp.Diff(x.(string), s))
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		if valY.Kind() == reflect.Slice || valY.Kind() == reflect.Array {
			child := make([]interface{}, valY.Len())
			for i := 0; i < valY.Len(); i++ {
				child[i] = valY.Index(i).Interface()
			}
			return isInSlice(valX, child...)
		}
		return isInSlice(valX, y)
	case reflect.Map:
		if valY.Kind() != reflect.Map {
			return d.Errorf("type mismatch -%T +%T", x, y)
		}
		return isInMap(valX, valY)
	}
	isEqual, s := Equal(x, y)
	if isEqual {
		return nil
	}
	return d.Errorf(s)
}

func isInMap(parent reflect.Value, child reflect.Value) *diff {
	d := newDiff()
	for _, key := range child.MapKeys() {
		p := parent.MapIndex(key)
		if !p.IsValid() {
			d.Missing(fmt.Sprintf("%v key=%v", parent.Type(), key))
			continue
		}
		c := child.MapIndex(key)
		d.Append(contains(p.Interface(), c.Interface()))
	}
	return d
}

func isInSlice(parent reflect.Value, child ...interface{}) *diff {
	d := newDiff()
	for _, v := range child {
		found := false
		for i := 0; i < parent.Len(); i++ {
			p := parent.Index(i)
			if contains(p.Interface(), v).Equal() {
				found = true
				break
			}
		}
		if !found {
			d.Missing(v)
		}
	}
	return d
}

// Equal use the cmp.Diff method to check equality and display differences.
// This method checks all unexpected values
func Equal(actual, expected interface{}) (bool, string) {
	/* var opts []cmp.Option
	t := reflect.TypeOf(actual)
	if t == nil {
	} else if t.Kind() == reflect.Struct {
		opts = append(opts, cmp.AllowUnexported(actual))
	} else if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		opts = append(opts, cmp.AllowUnexported(reflect.ValueOf(actual).Elem().Interface()))
	}
	t = reflect.TypeOf(expected)
	if t == nil {
	} else if t.Kind() == reflect.Struct {
		opts = append(opts, cmp.AllowUnexported(expected))
	} else if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		opts = append(opts, cmp.AllowUnexported(reflect.ValueOf(expected).Elem().Interface()))
	} */
	opts := allowUnexported(actual)

	r := cmp.Diff(actual, expected, opts...)
	return r == "", r
}

// allowUnexported sets up i to be compared including unexported fields using cmp.Diff or cmp.Equal.
// this function includes all unexported embedded structs or pointers to structs at all depths
func allowUnexported(i interface{}) []cmp.Option {
	opts := make([]cmp.Option, 0)
	t := reflect.TypeOf(i)
	// skip invalid types
	if t == nil {
		return opts
	}
	// add struct and pointers to struct
	switch t.Kind() {
	case reflect.Ptr:
		if t.Elem().Kind() != reflect.Struct {
			return opts
		}
		if reflect.ValueOf(i).IsNil() {
			return opts
		}
		i = reflect.ValueOf(i).Elem().Interface()
		fallthrough
	case reflect.Struct:
		opts = append(opts, cmp.AllowUnexported(i))
		rStruct := reflect.ValueOf(i)

		// look through all fields of a struct for embedded structs
		for index := 0; index < rStruct.NumField(); index++ {
			v := rStruct.Field(index)
			if v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct {
				// to support unexported (private) fields we need to create a copy
				// of the field and then dereference the pointer to that struct
				i = reflect.New(v.Elem().Type()).Elem().Interface()
				opts = append(opts, allowUnexported(i)...)
				continue
			}
			if !v.CanInterface() {
				// if the field is unexported (private) we wouldn't be able
				// get the interface{} so instead create a copy of that field
				v = reflect.New(v.Type()).Elem()
			}
			opts = append(opts, allowUnexported(v.Interface())...)
		}
	case reflect.Map:
		m := reflect.ValueOf(i)
		for _, key := range m.MapKeys() {
			v := m.MapIndex(key)
			opts = append(opts, allowUnexported(v.Interface())...)
		}
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		s := reflect.ValueOf(i)
		for i := 0; i < s.Len(); i++ {
			v := s.Index(i)
			opts = append(opts, allowUnexported(v.Interface())...)
		}
	default:
		return opts
	}

	return opts
}

// CmpFuncs tries to determine if x is the same function as y.
func CmpFuncs(x, y interface{}) (b bool, s string) {
	if x == nil || y == nil {
		if x == y {
			return true, ""
		}
		return false, fmt.Sprintf("%v != %v", x, y)
	}

	valX := reflect.ValueOf(x)
	valY := reflect.ValueOf(y)

	if valX.Kind() != reflect.Func || valY.Kind() != reflect.Func {
		return false, fmt.Sprintf("can only compare functions x=%v(%v) y=%v(%v) ", valX.Type(), x, valY.Type(), y)
	}

	if valY.Pointer() == valX.Pointer() {
		return true, ""
	}
	return false, fmt.Sprintf("funcs not equal 0x%x != 0x%x", valY.Pointer(), valX.Pointer())
}

func newDiff() *diff {
	return &diff{
		plus:  make([]interface{}, 0),
		minus: make([]interface{}, 0),
		msgs:  make([]string, 0),
	}
}

type diff struct {
	// values that are in y not x
	plus []interface{}
	// values that are in x not y
	minus []interface{}
	// msgs is used for additional messaging
	msgs []string
}

func (d *diff) Errorf(format string, values ...interface{}) *diff {
	d.msgs = append(d.msgs, fmt.Sprintf(format, values...))
	return d
}

func (d *diff) Extra(i interface{}) {
	d.plus = append(d.plus, i)
}

func (d *diff) Missing(i interface{}) {
	d.minus = append(d.minus, i)
}

func (d *diff) Equal() bool {
	if d == nil {
		return true
	}
	return len(d.plus) == 0 && len(d.minus) == 0 && len(d.msgs) == 0
}

func (d *diff) Append(v *diff) {
	if v == nil {
		return
	}
	d.msgs = append(d.msgs, v.msgs...)
	d.plus = append(d.plus, v.plus...)
	d.minus = append(d.minus, v.minus...)
}

func (d *diff) String() (s string) {
	if d == nil {
		return ""
	}
	if len(d.msgs) > 0 {
		for _, v := range d.msgs {
			s += v + "\n"
		}
		return s
	}

	if len(d.plus) > 0 {
		s = "+"
		for _, v := range d.plus {
			s += fmt.Sprintf("%v\n", v)
		}
	}

	if len(d.minus) > 0 {
		s += "-"
		for _, v := range d.minus {
			s += fmt.Sprintf("%v\n", v)
		}
	}
	return s
}
