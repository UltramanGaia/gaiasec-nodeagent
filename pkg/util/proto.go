package util

import (
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// SanitizeProtoStrings normalizes all protobuf string fields to valid UTF-8 in place.
func SanitizeProtoStrings(message proto.Message) {
	if message == nil {
		return
	}

	sanitizeProtoMessage(message.ProtoReflect())
}

func sanitizeProtoMessage(message protoreflect.Message) {
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch {
		case field.IsList():
			sanitizeList(message, field, value.List())
		case field.IsMap():
			sanitizeMap(message, field, value.Map())
		case field.Kind() == protoreflect.StringKind:
			sanitizeStringField(message, field, value.String())
		case field.Kind() == protoreflect.MessageKind:
			sanitizeProtoMessage(value.Message())
		}

		return true
	})
}

func sanitizeList(message protoreflect.Message, field protoreflect.FieldDescriptor, list protoreflect.List) {
	switch field.Kind() {
	case protoreflect.StringKind:
		for i := 0; i < list.Len(); i++ {
			list.Set(i, protoreflect.ValueOfString(sanitizeUTF8String(list.Get(i).String())))
		}
	case protoreflect.MessageKind:
		for i := 0; i < list.Len(); i++ {
			sanitizeProtoMessage(list.Get(i).Message())
		}
	}
}

func sanitizeMap(message protoreflect.Message, field protoreflect.FieldDescriptor, mapValue protoreflect.Map) {
	keyDesc := field.MapKey()
	valueDesc := field.MapValue()

	if keyDesc.Kind() == protoreflect.StringKind {
		entries := make([]struct {
			key   protoreflect.MapKey
			value protoreflect.Value
		}, 0, mapValue.Len())
		originalKeys := make([]protoreflect.MapKey, 0, mapValue.Len())

		mapValue.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			originalKeys = append(originalKeys, key)
			if valueDesc.Kind() == protoreflect.MessageKind {
				sanitizeProtoMessage(value.Message())
			}
			if valueDesc.Kind() == protoreflect.StringKind {
				value = protoreflect.ValueOfString(sanitizeUTF8String(value.String()))
			}

			entries = append(entries, struct {
				key   protoreflect.MapKey
				value protoreflect.Value
			}{
				key:   protoreflect.ValueOfString(sanitizeUTF8String(key.String())).MapKey(),
				value: value,
			})

			return true
		})

		for _, key := range originalKeys {
			mapValue.Clear(key)
		}
		for _, entry := range entries {
			mapValue.Set(entry.key, entry.value)
		}
		return
	}

	if valueDesc.Kind() == protoreflect.StringKind {
		mapValue.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			mapValue.Set(key, protoreflect.ValueOfString(sanitizeUTF8String(value.String())))
			return true
		})
		return
	}

	if valueDesc.Kind() == protoreflect.MessageKind {
		mapValue.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			sanitizeProtoMessage(value.Message())
			return true
		})
	}
}

func sanitizeStringField(message protoreflect.Message, field protoreflect.FieldDescriptor, value string) {
	message.Set(field, protoreflect.ValueOfString(sanitizeUTF8String(value)))
}

func sanitizeUTF8String(value string) string {
	return strings.ToValidUTF8(value, "")
}
