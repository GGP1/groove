package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	cases := []struct {
		desc string
		path string
	}{
		{desc: "Env var path", path: "testdata/mock_config.yaml"},
		{desc: "Home directory", path: ""},
	}

	expectedDev := true
	expectedDBHost := "localhost"
	expectedDBPort := 5432

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			os.Setenv("GROOVE_CONFIG", tc.path)

			_, err := New()
			assert.NoError(t, err)

			gotDev := viper.Get("development")
			assert.Equal(t, expectedDev, gotDev)

			gotDBHost := viper.Get("postgres.host")
			assert.Equal(t, expectedDBHost, gotDBHost)

			gotDBPort := viper.Get("postgres.port")
			assert.Equal(t, expectedDBPort, gotDBPort)
		})
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if os.RemoveAll(filepath.Join(home, ".groove")); err != nil {
		t.Fatal(err)
	}
}

func TestNewErrors(t *testing.T) {
	cases := []struct {
		set  func()
		desc string
		path string
	}{
		{
			desc: "Invalid path",
			path: "invalid_file.yaml",
			set:  func() {},
		},
		{
			desc: "Invalid extension",
			path: "testdata/mock_config",
			set:  func() {},
		},
		{
			desc: "Home path error",
			path: "",
			set:  func() {},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.set()
			if tc.path == "" {
				env := "HOME"
				switch runtime.GOOS {
				case "windows":
					env = "USERPROFILE"
				case "plan9":
					env = "home"
				}
				os.Setenv(env, "")
			}

			os.Setenv("groove_CONFIG", tc.path)

			_, err := New()
			assert.Error(t, err)
		})
	}
}
