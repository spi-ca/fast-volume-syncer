package util

import (
	flags "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"strings"
)

// pflagValue is a wrapper aroung *flags.flag
// that implements FlagValue
type pflagViperValue struct {
	flag *flags.Flag
}

// HasChanged returns whether the flag has changes or not.
func (p pflagViperValue) HasChanged() bool {
	return p.flag.Changed
}

// Name returns the name of the flag.
func (p pflagViperValue) Name() string {
	return p.flag.Name
}

// ValueString returns the value of the flag as a string.
func (p pflagViperValue) ValueString() string {
	return p.flag.Value.String()
}

// ValueType returns the type of the flag as a string.
func (p pflagViperValue) ValueType() string {
	return p.flag.Value.Type()
}

type PFlagViperReplacer struct {
	*flags.FlagSet
	Replacer *strings.Replacer
}

// VisitAll iterates over all *flags.Flag inside the *flags.FlagSet.
func (p PFlagViperReplacer) VisitAll(fn func(flag viper.FlagValue)) {
	p.FlagSet.VisitAll(p.bindPFlagToViper)
}

func (r PFlagViperReplacer) bindPFlagToViper(flag *flags.Flag) {
	_ = viper.BindFlagValue(r.Replacer.Replace(flag.Name), pflagViperValue{flag})
}
