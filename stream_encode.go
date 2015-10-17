package yaml

import (
	"fmt"
	"reflect"
	"sort"
)

type StreamEncoder struct {
	*encoder
}

// Basic lifecycle functions
func NewStreamEncoder() *StreamEncoder {
	rawEnc := newEncoder()
	return &StreamEncoder{rawEnc}
}

func (enc *StreamEncoder) Finish() []byte {
	defer enc.destroy()
	enc.finish()
	return enc.out
}

type MappingIteratorFunc func(key, value reflect.Value, flow bool, wouldOmit bool) error

func iterStruct(in reflect.Value, itemFunc MappingIteratorFunc) error {
	sinfo, err := getStructInfo(in.Type())
	if err != nil {
		panic(err)
	}

	for _, info := range sinfo.FieldsList {
		var value reflect.Value
		if info.Inline == nil {
			value = in.Field(info.Num)
		} else {
			value = in.FieldByIndex(info.Inline)
		}
		wouldOmit := info.OmitEmpty && isZero(value)

		if err := itemFunc(reflect.ValueOf(info.Key), value, info.Flow, wouldOmit); err != nil {
			return err
		}
	}
	if sinfo.InlineMap >= 0 {
		m := in.Field(sinfo.InlineMap)
		if m.Len() > 0 {
			keys := keyList(m.MapKeys())
			sort.Sort(keys)
			for _, k := range keys {
				if _, found := sinfo.FieldsMap[k.String()]; found {
					panic(fmt.Sprintf("Can't have key %q in inlined map; conflicts with struct field", k.String()))
				}
				if err := itemFunc(k, m.MapIndex(k), false, false); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// helper to make marshaling structs and maps more convinient
func (enc *StreamEncoder) EmitAsMap(tag string, flow bool, mapping reflect.Value, emitItem MappingIteratorFunc) error {
	if !mapping.IsValid() {
		enc.nilv()
		return nil
	}
	if mapping.Kind() == reflect.Ptr {
		if mapping.IsNil() {
			enc.nilv()
			return nil
		} else {
			mapping = mapping.Elem()
		}
	}
	enc.BeginMapping("", tag, flow)

	switch mapping.Kind() {
	case reflect.Struct:
		iterStruct(mapping, emitItem)
	case reflect.Map:
		keys := keyList(mapping.MapKeys())
		sort.Sort(keys)
		for _, k := range keys {
			if err := emitItem(k, mapping.MapIndex(k), false, false); err != nil {
				return err
			}
		}
	case reflect.Slice:
		if mapping.Type().Elem() == mapItemType {
			slice := mapping.Convert(reflect.TypeOf([]MapItem{})).Interface().([]MapItem)
			for _, item := range slice {
				if err := emitItem(reflect.ValueOf(item.Key), reflect.ValueOf(item.Value), false, false); err != nil {
					return err
				}
			}
		} else {
			panic("Cannot marshal type " + mapping.Type().String() + " as a map")
		}
	default:
		panic("Cannot marshal type " + mapping.Type().String() + " as a map")
	}

	enc.EndMapping()

	return nil
}

// collections helpers
func (enc *StreamEncoder) BeginMapping(anchor, tag string, flow bool) {
	style := yaml_BLOCK_MAPPING_STYLE
	if flow {
		style = yaml_FLOW_MAPPING_STYLE
	}

	enc.must(yaml_mapping_start_event_initialize(&enc.event, []byte(anchor), []byte(tag), tag == "", style))
	enc.emit()
}

func (enc *StreamEncoder) EndMapping() {
	enc.must(yaml_mapping_end_event_initialize(&enc.event))
	enc.emit()
}

func (enc *StreamEncoder) BeginSequence(anchor, tag string, flow bool) {
	style := yaml_BLOCK_SEQUENCE_STYLE
	if flow {
		style = yaml_FLOW_SEQUENCE_STYLE
	}

	enc.must(yaml_sequence_start_event_initialize(&enc.event, []byte(anchor), []byte(tag), tag == "", style))
	enc.emit()
}

func (enc *StreamEncoder) EndSequence() {
	enc.must(yaml_sequence_end_event_initialize(&enc.event))
	enc.emit()
}

// access to existing marshal functionality
func (enc *StreamEncoder) EmitValue(tag string, o reflect.Value, flow bool) {
	prevFlow := enc.flow
	enc.flow = flow
	enc.marshal(tag, o)
	enc.flow = prevFlow
}

func (enc *StreamEncoder) EmitComment(comment string, ownLine bool) {
	enc.must(yaml_comment_event_initialize(&enc.event, []byte(comment), ownLine))
	enc.emit()
}
