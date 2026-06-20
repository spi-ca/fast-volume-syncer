// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import (
	"strings"

	flags "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// pflagViperValue adapts a pflag.Flag to Viper's FlagValue interface.
type pflagViperValue struct {
	// flag is the wrapped pflag definition.
	flag *flags.Flag
}

// HasChanged reports whether the wrapped flag was set explicitly.
func (p pflagViperValue) HasChanged() bool {
	return p.flag.Changed
}

// Name returns the original pflag name.
func (p pflagViperValue) Name() string {
	return p.flag.Name
}

// ValueString returns the wrapped flag value in string form.
func (p pflagViperValue) ValueString() string {
	return p.flag.Value.String()
}

// ValueType returns the wrapped pflag value type name.
func (p pflagViperValue) ValueType() string {
	return p.flag.Value.Type()
}

// PFlagViperReplacer exposes a FlagSet to Viper after rewriting flag names, such as kebab-case to dotted keys.
type PFlagViperReplacer struct {
	// FlagSet supplies the flags that will be rebound into Viper.
	*flags.FlagSet
	// Replacer rewrites pflag names into the Viper key format.
	Replacer *strings.Replacer
}

// VisitAll walks every flag and binds it into Viper under its rewritten key.
func (p PFlagViperReplacer) VisitAll(fn func(flag viper.FlagValue)) {
	p.FlagSet.VisitAll(p.bindPFlagToViper)
}

// bindPFlagToViper binds one flag under the Viper key produced by Replacer.
func (r PFlagViperReplacer) bindPFlagToViper(flag *flags.Flag) {
	_ = viper.BindFlagValue(r.Replacer.Replace(flag.Name), pflagViperValue{flag})
}
