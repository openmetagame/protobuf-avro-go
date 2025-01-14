package protoavro

import (
	"fmt"
	"sort"

	"go.einride.tech/protobuf-avro/avro"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (s schemaInferrer) inferMapSchema(field protoreflect.FieldDescriptor, recursiveIndex int) (avro.Schema, error) {
	fieldKind, err := s.inferFieldKind(field, recursiveIndex)
	if err != nil {
		return nil, err
	}
	return avro.Nullable(avro.Array{
		Type:  avro.ArrayType,
		Items: fieldKind,
	}), nil
}

func (o *SchemaOptions) encodeMap(
	field protoreflect.FieldDescriptor,
	m protoreflect.Map,
	recursiveIndex int,
) (interface{}, error) {
	// m.Range ranges over the entries in unspecified order.
	// To aid in testing, the keys are sorted. This is similar
	// to what json.Marshal does for maps.
	keys := make([]protoreflect.MapKey, 0, m.Len())
	m.Range(func(key protoreflect.MapKey, _ protoreflect.Value) bool {
		keys = append(keys, key)
		return true
	})
	sort.Slice(keys, func(i, j int) bool {
		// key.String will return a string for any key type (not just strings)
		// for example 1 would be "1"
		return keys[i].String() < keys[j].String()
	})

	entries := make([]interface{}, 0, m.Len())
	valueField := field.MapValue()
	keyField := field.MapKey()
	for _, key := range keys {
		value := m.Get(key)
		keyValue, err := o.fieldKindJSON(keyField, key.Value(), recursiveIndex, true)
		if err != nil {
			return nil, err
		}
		valueValue, err := o.fieldKindJSON(valueField, value, recursiveIndex, true)
		if err != nil {
			return nil, err
		}
		entries = append(entries, map[string]interface{}{
			"key":   keyValue,
			"value": valueValue,
		})
	}
	return o.unionValue("array", entries), nil
}

func (o SchemaOptions) decodeMap(data interface{}, f protoreflect.FieldDescriptor, mp protoreflect.Map) error {
	list, err := decodeListLike(data, "array")
	if err != nil {
		return err
	}
	return o.decodeMapEntries(list, f, mp)
}

func (o SchemaOptions) decodeMapEntries(data []interface{}, f protoreflect.FieldDescriptor, mp protoreflect.Map) error {
	for _, el := range data {
		entry, ok := el.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected map entry, got %T for '%s'", el, f.Name())
		}
		keyData, ok := entry["key"]
		if !ok {
			return fmt.Errorf("missing 'key' in map entry for '%s'", f.Name())
		}
		valueData, ok := entry["value"]
		if !ok {
			return fmt.Errorf("missing 'value' in map entry for '%s'", f.Name())
		}
		keyValue, err := o.decodeFieldKind(keyData, protoreflect.Value{}, f.MapKey())
		if err != nil {
			return err
		}
		valueValue, err := o.decodeFieldKind(valueData, mp.NewValue(), f.MapValue())
		if err != nil {
			return err
		}
		mp.Set(keyValue.MapKey(), valueValue)
	}
	return nil
}
