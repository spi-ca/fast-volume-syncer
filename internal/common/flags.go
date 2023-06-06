package common

import (
	flags "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"strings"
)

// pflagValue is a wrapper aroung *flags.flag
// that implements FlagValue
type flagValue struct {
	flag *flags.Flag
}

// HasChanged returns whether the flag has changes or not.
func (p flagValue) HasChanged() bool {
	return p.flag.Changed
}

// Name returns the name of the flag.
func (p flagValue) Name() string {
	return p.flag.Name
}

// ValueString returns the value of the flag as a string.
func (p flagValue) ValueString() string {
	return p.flag.Value.String()
}

// ValueType returns the type of the flag as a string.
func (p flagValue) ValueType() string {
	return p.flag.Value.Type()
}

type PFlagReplacer struct {
	*flags.FlagSet
	Replacer *strings.Replacer
}

// VisitAll iterates over all *flags.Flag inside the *flags.FlagSet.
func (p PFlagReplacer) VisitAll(fn func(flag viper.FlagValue)) {
	p.FlagSet.VisitAll(p.bindPFlagToViper)
}

func (r PFlagReplacer) bindPFlagToViper(flag *flags.Flag) {
	_ = viper.BindFlagValue(r.Replacer.Replace(flag.Name), flagValue{flag})
}
