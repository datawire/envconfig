package envconfig_test

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/envconfig"
)

// Note: DO NOT use t.Parallel(); because these tests all make use of
// the global environment (os.Getenv/os.Setenv), they are not safe to
// run in parallel.

func TestAbsoluteURL(t *testing.T) {
	var config struct {
		U *url.URL `env:"CONFIG_URL,parser=absolute-URL"`
	}
	parser, err := envconfig.GenerateParser(reflect.TypeOf(config), nil)
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		Input       string
		ExpectError bool
	}{
		{Input: "https://api.example.com/"},
		{Input: "localhost:8080", ExpectError: true},
		{Input: "file:///home/user/repo.git"}, // a valid absolute-URL with an empty host-part
		{Input: "/home/user/repo.git", ExpectError: true},
	}
	for i, tc := range testcases {
		tc := tc // capture loop variable
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			config.U = nil
			t.Setenv("CONFIG_URL", tc.Input)

			warn, fatal := parser.ParseFromEnv(&config)

			assert.Equal(t, len(warn), 0, "There should be no warnings")
			if tc.ExpectError {
				assert.Equal(t, len(fatal), 1, "There should be 1 fatal error")
				assert.Nil(t, config.U, "config.U should be nil because there should be an error")
			} else {
				assert.Equal(t, len(fatal), 0, "There should be no fatal errors")
				if !assert.NotNil(t, config.U, "config.U should not be nil") {
					return
				}
				assert.Equal(t, config.U.String(), tc.Input, "config.U should stringify to the input")
			}
		})
	}
}

func TestRecursive(t *testing.T) {
	var config struct {
		ParentThing string `env:"PARENT_THING,parser=nonempty-string"`
		Child       struct {
			Thing1 string `env:"CHILD_THING1,parser=nonempty-string"`
			Thing2 string `env:"CHILD_THING2,parser=nonempty-string"`
		}
	}
	parser, err := envconfig.GenerateParser(reflect.TypeOf(config), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PARENT_THING", "foo")
	t.Setenv("CHILD_THING1", "bar")
	t.Setenv("CHILD_THING2", "baz")
	warn, fatal := parser.ParseFromEnv(&config)
	assert.Equal(t, len(warn), 0, "There should be no warnings")
	assert.Equal(t, len(fatal), 0, "There should be no errors")
	assert.Equal(t, config.ParentThing, "foo")
	assert.Equal(t, config.Child.Thing1, "bar")
	assert.Equal(t, config.Child.Thing2, "baz")
}

func TestSmokeTestAllParsers(t *testing.T) {
	type testcase struct {
		Object   interface{}
		EnvVar   string
		Expected string
		Errors   int
		Warnings int
	}
	// This isn't going in to any depth on any of the types; just
	// checking that the parser and setter don't panic.
	tests := map[string]map[string]testcase{
		"string": {
			"nonempty-string": {
				Object: &struct {
					Value string `env:"VALUE,parser=nonempty-string"`
				}{},
				EnvVar:   "str",
				Expected: `&{str}`,
			},
			"nonempty-string-unset": {
				// Error, required value with unset environment variable,
				Object: &struct {
					Value string `env:"UNSET_VALUE,parser=nonempty-string"`
				}{},
				Errors:   1,
				Expected: `&{}`,
			},
			"nonempty-string-default-set": {
				// Parser errors on empty string and falls back to default
				Object: &struct {
					Value string `env:"VALUE,parser=nonempty-string,default=str"`
				}{},
				EnvVar:   "",
				Expected: `&{str}`,
				Warnings: 1,
			},
			"nonempty-string-default-unset": {
				// UNSET_VALUE is not present so parser called with default
				Object: &struct {
					Value string `env:"UNSET_VALUE,parser=nonempty-string,default=str"`
				}{},
				Expected: `&{str}`,
			},
			"possibly-empty-string": {
				Object: &struct {
					Value string `env:"VALUE,parser=possibly-empty-string"`
				}{},
				EnvVar:   "",
				Expected: `&{}`,
			},
			"possibly-empty-string-unset": {
				Object: &struct {
					Value string `env:"UNSET_VALUE,parser=possibly-empty-string"`
				}{},
				Expected: `&{}`,
				Errors:   1,
			},
			"possibly-empty-string-default-set": {
				Object: &struct {
					Value string `env:"VALUE,parser=possibly-empty-string,default=str"`
				}{},
				EnvVar:   "",
				Expected: `&{}`,
			},
			"possibly-empty-string-default-unset": {
				Object: &struct {
					// Use UNSET_VALUE to reference a non-existent env variable.
					Value string `env:"UNSET_VALUE,parser=possibly-empty-string,default=str"`
				}{},
				Expected: `&{str}`,
			},
			"logrus.ParseLevel": {
				Object: &struct {
					Value string `env:"VALUE,parser=logrus.ParseLevel"`
				}{},
				EnvVar:   "info",
				Expected: `&{info}`,
			},
		},
		"bool": {
			"empty/nonempty": {
				Object: &struct {
					Value bool `env:"VALUE,parser=empty/nonempty"`
				}{},
				EnvVar:   "false",
				Expected: `&{true}`,
			},
			"strconv.ParseBool": {
				Object: &struct {
					Value bool `env:"VALUE,parser=strconv.ParseBool"`
				}{},
				EnvVar:   "false",
				Expected: `&{false}`,
			},
		},
		"int": {
			"strconv.ParseInt": {
				Object: &struct {
					Value int `env:"VALUE,parser=strconv.ParseInt"`
				}{},
				EnvVar:   "123",
				Expected: `&{123}`,
			},
		},
		"int64": {
			"strconv.ParseInt": {
				Object: &struct {
					Value int64 `env:"VALUE,parser=strconv.ParseInt"`
				}{},
				EnvVar:   "123",
				Expected: `&{123}`,
			},
		},
		"float32": {
			"strconv.ParseFloat": {
				Object: &struct {
					Value float32 `env:"VALUE,parser=strconv.ParseFloat"`
				}{},
				EnvVar:   "12.52",
				Expected: "&{12.52}",
			},
		},
		"*url.URL": {
			"absolute-URL": {
				Object: &struct {
					Value *url.URL `env:"VALUE,parser=absolute-URL"`
				}{},
				EnvVar:   "https://example.com/",
				Expected: `&{https://example.com/}`,
			},
		},
		"time.Duration": {
			"integer-seconds": {
				Object: &struct {
					Value time.Duration `env:"VALUE,parser=integer-seconds"`
				}{},
				EnvVar:   "182",
				Expected: `&{3m2s}`,
			},
			"time.ParseDuration": {
				Object: &struct {
					Value time.Duration `env:"VALUE,parser=time.ParseDuration"`
				}{},
				EnvVar:   "3m2s",
				Expected: `&{3m2s}`,
			},
		},
	}

	for typeName, typetests := range tests {
		typetests := typetests
		t.Run(typeName, func(t *testing.T) {
			for parserName, testinfo := range typetests {
				testinfo := testinfo
				t.Run(parserName, func(t *testing.T) {
					parser, err := envconfig.GenerateParser(reflect.TypeOf(testinfo.Object).Elem(), nil)
					if err != nil {
						t.Fatal(err)
					}
					t.Setenv("VALUE", testinfo.EnvVar)
					warn, fatal := parser.ParseFromEnv(testinfo.Object)
					assert.Equalf(t, testinfo.Warnings, len(warn), "There should be %d warnings", testinfo.Warnings)
					assert.Equalf(t, testinfo.Errors, len(fatal), "There should be %d errors", testinfo.Errors)
					assert.Equal(t, testinfo.Expected, fmt.Sprintf("%v", testinfo.Object))
				})
			}
		})
	}

	for reflectType, typeHandler := range envconfig.DefaultFieldTypeHandlers() {
		typeName := reflectType.String()
		if len(tests[typeName]) == 0 {
			t.Errorf("no tests for type %q", typeName)
			continue
		}

		for parserName := range typeHandler.Parsers {
			if _, ok := tests[typeName][parserName]; !ok {
				t.Errorf("no test for type %q parser %q", typeName, parserName)
			}
		}
	}
}
