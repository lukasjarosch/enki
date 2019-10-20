package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// parseFlags parses the flags passed to the binary.
// If an error occurs, the error is logged and the help displayed.
func parseFlags(fs *pflag.FlagSet) {
	err := fs.Parse(os.Args[1:])
	switch {
	case err == pflag.ErrHelp:
		os.Exit(0)
	case err != nil:
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n\n", err.Error())
		fs.PrintDefaults()
		os.Exit(2)
	}
}

// bindFlags will bind the configured flags to environment variables, prefixed with 'EnvPrefix'.
// It will also set the 'hostname' field in viper in case it's needed.
func bindFlags(fs *pflag.FlagSet) {
	_ = viper.BindPFlags(fs)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}


func ParseFlagSet(set *pflag.FlagSet) {
	parseFlags(set)
	bindFlags(set)
}
