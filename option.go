package clic

import (
	"fmt"
	"sort"
)

// Option is a reusable option definition and its execution-local state.
type Option struct {
	Meta  OptionMeta
	Spec  OptionSpec
	State OptionState
}

// Options maps stable option keys to options.
type Options map[string]*Option

// OptionMeta is host-controlled option presentation and resolution metadata.
// Name defaults to the option's Spec.Key when it is empty.
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

type environmentTemplateData struct {
	EnvPrefix string
}

// Render expands Description templates with the supplied command metadata
// context. Option spellings remain literal because clic uses them to validate
// and resolve option values.
func (meta OptionMeta) Render(data MetadataTemplateData) (OptionMeta, error) {
	data.Name = meta.Name
	data.ShortName = meta.ShortName
	data.EnvName = meta.EnvName
	description, err := renderTemplateString(meta.Description, data)
	if err != nil {
		return OptionMeta{}, fmt.Errorf("render option description: %w", err)
	}
	meta.Description = description
	return meta, nil
}

// ResolveEnvName expands EnvName with its command's inherited environment
// prefix. EnvName templates receive only .EnvPrefix.
func (meta OptionMeta) ResolveEnvName(envPrefix string) (OptionMeta, error) {
	envName, err := renderTemplateString(meta.EnvName, environmentTemplateData{EnvPrefix: envPrefix})
	if err != nil {
		return OptionMeta{}, fmt.Errorf("render option environment name: %w", err)
	}
	meta.EnvName = envName
	return meta, nil
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

// Name returns the option's presentation name, defaulting to its stable key.
func (option Option) Name() string {
	if option.Meta.Name != "" {
		return option.Meta.Name
	}
	return option.Spec.Key
}

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
