package envconfig

import (
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// If you add something to this, please add to the
// TestSmokeTestAllParsers test.
var envConfigTypes = map[reflect.Type]fieldTypeHandler{

	// string
	reflect.TypeOf(""): {
		Parsers: map[string]func(string) (interface{}, error){
			"nonempty-string": func(str string) (interface{}, error) {
				if str == "" {
					return nil, ErrorNotSet
				}
				return str, nil
			},
			"possibly-empty-string": func(str string) (interface{}, error) { return str, nil },
			"logrus.ParseLevel": func(str string) (interface{}, error) {
				if _, err := logrus.ParseLevel(str); err != nil {
					return nil, err
				}
				return str, nil
			},
		},
		Setter: func(dst reflect.Value, src interface{}) { dst.SetString(src.(string)) },
	},

	// bool
	reflect.TypeOf(false): {
		Parsers: map[string]func(string) (interface{}, error){
			"empty/nonempty":    func(str string) (interface{}, error) { return str != "", nil },
			"strconv.ParseBool": func(str string) (interface{}, error) { return strconv.ParseBool(str) },
		},
		Setter: func(dst reflect.Value, src interface{}) { dst.SetBool(src.(bool)) },
	},

	// int
	reflect.TypeOf(int(0)): {
		Parsers: map[string]func(string) (interface{}, error){
			"strconv.ParseInt": func(str string) (interface{}, error) {
				i64, err := strconv.ParseInt(str, 10, 0)
				return int(i64), err
			},
		},
		Setter: func(dst reflect.Value, src interface{}) { dst.SetInt(int64(src.(int))) },
	},

	// int64
	reflect.TypeOf(int64(0)): {
		Parsers: map[string]func(string) (interface{}, error){
			"strconv.ParseInt": func(str string) (interface{}, error) { return strconv.ParseInt(str, 10, 64) },
		},
		Setter: func(dst reflect.Value, src interface{}) { dst.SetInt(src.(int64)) },
	},

	// *url.URL
	reflect.TypeOf((*url.URL)(nil)): {
		Parsers: map[string]func(string) (interface{}, error){
			"absolute-URL": func(str string) (interface{}, error) {
				u, err := url.Parse(str)
				if err != nil {
					return nil, err
				}
				isURL := strings.HasPrefix(u.String(), u.Scheme+"://") // as opposed to being a URN
				if !u.IsAbs() || !isURL {
					// Why do we need to check .IsAbs() _and_ isURL?  Because despite the
					// name, the `net/url` package is used for any URI; which means it can
					// be either a URL or a URN.  We need it to specifically be a URL, and
					// to reject a URN.
					//
					// Otherwise, "host:port", would parse as an absolute opaque URN, with
					// "scheme=host" and "opaque=port".
					return nil, errors.New("not an absolute URL")
				}
				return u, nil
			},
		},
		Setter: func(dst reflect.Value, src interface{}) { dst.Set(reflect.ValueOf(src.(*url.URL))) },
	},

	// time.Duration
	reflect.TypeOf(time.Duration(0)): {
		Parsers: map[string]func(string) (interface{}, error){
			"integer-seconds": func(str string) (interface{}, error) {
				secs, err := strconv.Atoi(str)
				if err != nil {
					return nil, err
				}
				return time.Duration(secs) * time.Second, nil
			},
			"time.ParseDuration": func(str string) (interface{}, error) { return time.ParseDuration(str) },
		},
		Setter: func(dst reflect.Value, src interface{}) { dst.SetInt(int64(src.(time.Duration))) },
	},
}
