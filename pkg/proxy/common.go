package proxy

import (
	"github.com/dustin/go-humanize"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

const (
	Prefix         = "pkg.proxy"
	UnmarshalKey   = "proxy"
	FormFile       = "file"
)

type Size uint64

// FromString set size in bytes from humanize string representation
func (s *Size) FromString(hSize string) Size {
	size, _ := humanize.ParseBytes(hSize)
	*s = Size(size)
	return *s
}

// StringToHumanizeSizeHookFunc returns decoder func hook for converting string representation to size in bytes
func StringToHumanizeSizeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(Size(0)) {
			return data, nil
		}
		var s Size
		return (&s).FromString(data.(string)), nil
	}
}
