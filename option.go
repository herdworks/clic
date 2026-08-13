package clic

import "sort"

// Option is a reusable option definition and its execution-local state.
type Option struct {
	Meta  OptionMeta
	Spec  OptionSpec
	State OptionState
}

// Options maps stable option keys to options.
type Options map[string]*Option

// OptionMeta is host-controlled option presentation and resolution metadata.
type OptionMeta struct {
	Name        string
	ShortName   string
	Description string
	EnvName     string
	Default     any
	Global      bool
	Hidden      bool
	ShowDefault bool
	Order       int
}

// OptionSpec is reusable option behavior.
type OptionSpec struct {
	Key      string
	Kind     OptionKind
	Required bool
}

// OptionState is execution-local parsed option state.
type OptionState struct{ Value any }

// OptionKind identifies an option value type.
type OptionKind uint8

const (
	StringOption OptionKind = iota
	BoolOption
)

// StringValue returns a string option's execution-local value.
func (option Option) StringValue() string { value, _ := option.State.Value.(string); return value }

// BoolValue returns a boolean option's execution-local value.
func (option Option) BoolValue() bool { value, _ := option.State.Value.(bool); return value }

// Sorted returns options in presentation order.
func (options Options) Sorted() []*Option {
	result := make([]*Option, 0, len(options))
	for _, option := range options {
		result = append(result, option)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Meta.Order != result[j].Meta.Order {
			return result[i].Meta.Order < result[j].Meta.Order
		}
		return result[i].Spec.Key < result[j].Spec.Key
	})
	return result
}

// MustOptionsFrom composes options returned from factories. It panics if a
// factory returns an empty key or a key already produced by another factory.
func MustOptionsFrom(factories ...func() Option) Options {
	options := make(Options, len(factories))
	for _, factory := range factories {
		option := factory()
		if option.Spec.Key == "" {
			panic("clic option factory returned an empty key")
		}
		if _, exists := options[option.Spec.Key]; exists {
			panic("clic option factories returned duplicate key " + option.Spec.Key)
		}
		options[option.Spec.Key] = &option
	}
	return options
}
