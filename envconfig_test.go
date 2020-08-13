package envconfig_test

import (
	"net/url"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/apro/cmd/amb-sidecar/types/internal/envconfig"
)

func TestAbsoluteRL(t *testing.T) {
	var config struct {
		U *url.URL `env:"CONFIG_URL,parser=absolute-URL"`
	}
	parser, err := envconfig.GenerateParser(reflect.TypeOf(config))
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
			os.Setenv("CONFIG_URL", tc.Input)

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
	parser, err := envconfig.GenerateParser(reflect.TypeOf(config))
	if err != nil {
		t.Fatal(err)
	}
	os.Setenv("PARENT_THING", "foo")
	os.Setenv("CHILD_THING1", "bar")
	os.Setenv("CHILD_THING2", "baz")
	warn, fatal := parser.ParseFromEnv(&config)
	assert.Equal(t, len(warn), 0, "There should be no warnings")
	assert.Equal(t, len(fatal), 0, "There should be no errors")
	assert.Equal(t, config.ParentThing, "foo")
	assert.Equal(t, config.Child.Thing1, "bar")
	assert.Equal(t, config.Child.Thing2, "baz")
}
