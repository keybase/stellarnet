package config

import (
	"fmt"
	"go/types"
	stdLog "log"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/strutils"
)

// ConfigOptions is a group of ConfigOptions that can be for convenience
// initialized and set at the same time.
type ConfigOptions []*ConfigOption

// Init calls Init on each ConfigOption passing on the cobra.Command.
func (cos ConfigOptions) Init(cmd *cobra.Command) error {
	for _, co := range cos {
		err := co.Init(cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

// Require calls Require on each ConfigOption.
func (cos ConfigOptions) Require() {
	for _, co := range cos {
		co.Require()
	}
}

// SetValues calls SetValue on each ConfigOption.
func (cos ConfigOptions) SetValues() {
	for _, co := range cos {
		co.SetValue()
	}
}

// ConfigOption is a complete description of the configuration of a command line option
type ConfigOption struct {
	Name           string              // e.g. "db-url"
	EnvVar         string              // e.g. "DATABASE_URL". Defaults to uppercase/underscore representation of name
	OptType        types.BasicKind     // The type of this option, e.g. types.Bool
	FlagDefault    interface{}         // A default if no option is provided. Omit or set to `nil` if no default
	Required       bool                // Whether this option must be set for Horizon to run
	Usage          string              // Help text
	CustomSetValue func(*ConfigOption) // Optional function for custom validation/transformation
	ConfigKey      interface{}         // Pointer to the final key in the linked Config struct
	flag           *pflag.Flag         // The persistent flag that the config option is attached to
}

// Init handles initialisation steps, including configuring and binding the env variable name.
func (co *ConfigOption) Init(cmd *cobra.Command) error {
	// Bind the command line and environment variable name
	// Unless overriden, default to a transform like tls-key -> TLS_KEY
	if co.EnvVar == "" {
		co.EnvVar = strutils.KebabToConstantCase(co.Name)
	}
	// Initialise and bind the persistent flags
	return co.setFlag(cmd)
}

// Bind binds the config option to viper.
func (co *ConfigOption) Bind() {
	viper.BindPFlag(co.Name, co.flag)
	viper.BindEnv(co.Name, co.EnvVar)
}

// Require checks that a required string configuration option is not empty, raising a user error if it is.
func (co *ConfigOption) Require() {
	co.Bind()
	if co.Required && viper.GetString(co.Name) == "" {
		stdLog.Fatalf("Invalid config: %s is blank. Please specify --%s on the command line or set the %s environment variable.", co.Name, co.Name, co.EnvVar)
	}
}

// SetValue sets a value in the global config, using a custom function, if one was provided.
func (co *ConfigOption) SetValue() {
	co.Bind()

	// Use a custom setting function, if one is provided
	if co.CustomSetValue != nil {
		co.CustomSetValue(co)
		// Otherwise, just set the provided arg directly
	} else if co.ConfigKey != nil {
		co.setSimpleValue()
	}
}

// UsageText returns the string to use for the usage text of the option. The
// string returned will be the Usage defined on the ConfigOption, along with
// the environment variable.
func (co *ConfigOption) UsageText() string {
	return fmt.Sprintf("%s (%s)", co.Usage, co.EnvVar)
}

// setSimpleValue sets the value of a ConfigOption's configKey, based on the ConfigOption's default type.
func (co *ConfigOption) setSimpleValue() {
	if co.ConfigKey != nil {
		switch co.OptType {
		case types.String:
			*(co.ConfigKey.(*string)) = viper.GetString(co.Name)
		case types.Int:
			*(co.ConfigKey.(*int)) = viper.GetInt(co.Name)
		case types.Bool:
			*(co.ConfigKey.(*bool)) = viper.GetBool(co.Name)
		case types.Uint:
			*(co.ConfigKey.(*uint)) = uint(viper.GetInt(co.Name))
		case types.Uint32:
			*(co.ConfigKey.(*uint32)) = uint32(viper.GetInt(co.Name))
		}
	}
}

// setFlag sets the correct pFlag type, based on the ConfigOption's default type.
func (co *ConfigOption) setFlag(cmd *cobra.Command) error {
	switch co.OptType {
	case types.String:
		// Set an empty string if no default was provided, since some value is always required for pflags
		if co.FlagDefault == nil {
			co.FlagDefault = ""
		}
		cmd.PersistentFlags().String(co.Name, co.FlagDefault.(string), co.UsageText())
	case types.Int:
		cmd.PersistentFlags().Int(co.Name, co.FlagDefault.(int), co.UsageText())
	case types.Bool:
		cmd.PersistentFlags().Bool(co.Name, co.FlagDefault.(bool), co.UsageText())
	case types.Uint:
		cmd.PersistentFlags().Uint(co.Name, co.FlagDefault.(uint), co.UsageText())
	case types.Uint32:
		cmd.PersistentFlags().Uint32(co.Name, co.FlagDefault.(uint32), co.UsageText())
	default:
		return errors.New("Unexpected OptType")
	}

	co.flag = cmd.PersistentFlags().Lookup(co.Name)

	return nil
}

// SetDuration converts a command line int to a duration, and stores it in the final config.
func SetDuration(co *ConfigOption) {
	*(co.ConfigKey.(*time.Duration)) = time.Duration(viper.GetInt(co.Name)) * time.Second
}

// SetURL converts a command line string to a URL, and stores it in the final config.
func SetURL(co *ConfigOption) {
	urlString := viper.GetString(co.Name)
	if urlString != "" {
		urlType, err := url.Parse(urlString)
		if err != nil {
			stdLog.Fatalf("Unable to parse URL: %s/%v", urlString, err)
		}
		*(co.ConfigKey.(**url.URL)) = urlType
	}
}
