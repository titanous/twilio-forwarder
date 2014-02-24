package underscore

import (
	"net/url"
	"reflect"
	"strconv"
)

var BoolFlag = []string{"false", "true"}

// Converting a struct into a url.Values map
func StructToMap(in interface{}) url.Values {
	val := make(url.Values)

	el := reflect.ValueOf(in).Elem()
	tp := el.Type()

	for i := 0; i < el.NumField(); i++ {

		k := tp.Field(i).Name
		f := el.Field(i)
		c := f.Interface()

		var v string
		var vv []string

		switch reflect.TypeOf(c).Kind() {

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v = strconv.FormatInt(f.Int(), 10)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v = strconv.FormatUint(f.Uint(), 10)

		case reflect.Float32:
			v = strconv.FormatFloat(f.Float(), 'f', 4, 32)

		case reflect.Float64:
			v = strconv.FormatFloat(f.Float(), 'f', 4, 64)

		case reflect.Bool:
			if f.Bool() {
				v = BoolFlag[1]
			} else {
				v = BoolFlag[0]
			}

		case reflect.String:
			v = f.String()

		case reflect.Slice:
			s := reflect.ValueOf(c)

			for x := 0; x < s.Len(); x++ {
				vv = append(vv, s.Index(x).String())
			}
		}

		if v != "" {
			val.Set(k, v)
		}

		for ix := range vv {
			val.Add(k, vv[ix])
		}
	}

	return val
}
