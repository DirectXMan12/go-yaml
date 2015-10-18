package yaml

import (
	"encoding/base64"
	"reflect"
	"strconv"
)

type StreamDecoder struct {
	*parser
}

type DecoderEventKind uint8
const (
	ScalarEvent DecoderEventKind = iota
	AliasEvent
	MappingStartEvent
	MappingEndEvent
	SequenceStartEvent
	SequenceEndEvent
	FinishEvent
)

func (k DecoderEventKind) String() string {
	switch k {
	case ScalarEvent:
		return "ScalarEvent"
	case AliasEvent:
		return "AliasEvent"
	case MappingStartEvent:
		return "MappingStartEvent"
	case MappingEndEvent:
		return "MappingEndEvent"
	case SequenceStartEvent:
		return "SequenceStartEvent"
	case SequenceEndEvent:
		return "SequenceEndEvent"
	case FinishEvent:
		return "FinishEvent"
	default:
		return "UnknownEvent"
	}
}

// TODO: add in style info (instead of just `Flow` as a bool)?
// TODO: add in directive info?
type Event struct {
	Kind           DecoderEventKind

	Line           int
	Column         int

	Value          reflect.Value
	Tag            string
	Anchor         string
	Implicit       bool
	QuotedImplicit bool
	Flow           bool
}


// Basic lifecycle functions
func NewStreamDecoder(input []byte) *StreamDecoder {
	parser := newParser(input)
	return &StreamDecoder{parser}
}

func decode_scalar(rawEvent *yaml_event_t, outEvent *Event) {
	var resolved interface{}
	var tag string
	if outEvent.Tag == "" && !rawEvent.implicit {
		resolved = string(rawEvent.value)
		tag = yaml_STR_TAG
	} else {
		tag, resolved = resolve(outEvent.Tag, string(rawEvent.value))
		if tag == yaml_BINARY_TAG {
			data, err := base64.StdEncoding.DecodeString(resolved.(string))
			if err != nil {
				failf("!!binary value contains invalid base64 data")
			}
			resolved = string(data)
		}
	}

	outEvent.Tag = tag
	outEvent.Value = reflect.ValueOf(resolved)
}

func (dec *StreamDecoder) NextEvent() Event {
	evt := Event{
		Line:           dec.event.start_mark.line,
		Column:         dec.event.start_mark.column,
		Anchor:         string(dec.event.anchor),
		Tag:            string(dec.event.tag),
		Implicit:       dec.event.implicit,
		QuotedImplicit: dec.event.quoted_implicit,
	}

	switch dec.event.typ {
	case yaml_SCALAR_EVENT:
		evt.Kind = ScalarEvent
		evt.Flow = (dec.event.style != yaml_style_t(yaml_LITERAL_SCALAR_STYLE) && dec.event.style != yaml_style_t(yaml_FOLDED_SCALAR_STYLE))
		decode_scalar(&dec.event, &evt)
	case yaml_ALIAS_EVENT:
		evt.Kind = AliasEvent
	case yaml_MAPPING_START_EVENT:
		evt.Kind = MappingStartEvent
		evt.Flow = dec.event.style == yaml_style_t(yaml_FLOW_MAPPING_STYLE)
	case yaml_MAPPING_END_EVENT:
		evt.Kind = MappingEndEvent
	case yaml_SEQUENCE_START_EVENT:
		evt.Kind = SequenceStartEvent
		evt.Flow = dec.event.style == yaml_style_t(yaml_FLOW_SEQUENCE_STYLE)
	case yaml_SEQUENCE_END_EVENT:
		evt.Kind = SequenceEndEvent
	case yaml_DOCUMENT_END_EVENT:
		evt.Kind = FinishEvent
		defer dec.destroy()
	case yaml_DOCUMENT_START_EVENT:
		dec.skip()
		return dec.NextEvent()
	case yaml_STREAM_END_EVENT:
		evt.Kind = FinishEvent
		defer dec.destroy()
		return evt // don't skip -- nothing comes after a stream end event
	default:
		panic("attempted to parse unknown event: " + strconv.Itoa(int(dec.event.typ)))
	}

	dec.skip()
	return evt
}
