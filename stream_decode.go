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
	DocumentStartEvent
	DocumentEndEvent
	FinishEvent
	CommentEvent
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
	case DocumentStartEvent:
		return "DocumentStartEvent"
	case DocumentEndEvent:
		return "DocumentEndEvent"
	case FinishEvent:
		return "FinishEvent"
	case CommentEvent:
		return "CommentEvent"
	default:
		return "UnknownEvent"
	}
}

type ScalarStyleKind uint8
const (
	AnyStyle = iota
	PlainScalar
	SingleQuotedScalar
	DoubleQuotedScalar
	LiteralBlockScalar
	FoldedBlockScalar
)

func (s ScalarStyleKind) String() string {
	switch s {
	case AnyStyle:
		return "AnyStyle"
	case PlainScalar:
		return "PlainScalar"
	case SingleQuotedScalar:
		return "SingleQuotedScalar"
	case DoubleQuotedScalar:
		return "DoubleQuotedScalar"
	case LiteralBlockScalar:
		return "LiteralBlockScalar"
	case FoldedBlockScalar:
		return "FoldedBlockScalar"
	default:
		return "UnknownScalarStyle"
	}
}

func (inStyle ScalarStyleKind) toYamlScalarStyle() yaml_scalar_style_t {
	switch inStyle {
	case PlainScalar:
		return yaml_PLAIN_SCALAR_STYLE
	case SingleQuotedScalar:
		return yaml_SINGLE_QUOTED_SCALAR_STYLE
	case DoubleQuotedScalar:
		return yaml_DOUBLE_QUOTED_SCALAR_STYLE
	case LiteralBlockScalar:
		return yaml_LITERAL_SCALAR_STYLE
	case FoldedBlockScalar:
		return yaml_LITERAL_SCALAR_STYLE
	default:
		return yaml_ANY_SCALAR_STYLE
	}
}

func (inStyle yaml_scalar_style_t) toScalarStyleKind() ScalarStyleKind {
	switch inStyle {
	case yaml_PLAIN_SCALAR_STYLE:
		return PlainScalar
	case yaml_SINGLE_QUOTED_SCALAR_STYLE:
		return SingleQuotedScalar
	case yaml_DOUBLE_QUOTED_SCALAR_STYLE:
		return DoubleQuotedScalar
	case yaml_LITERAL_SCALAR_STYLE:
		return LiteralBlockScalar
	case yaml_FOLDED_SCALAR_STYLE:
		return FoldedBlockScalar
	default:
		return AnyStyle
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
	ScalarStyle    ScalarStyleKind

	YAMLVersion    *VersionInfo
	TagDefinitions []TagInfo
}

type VersionInfo struct {
	Major int8
	Minor int8
}

type TagInfo struct {
	Handle string
	Prefix string
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
		evt.ScalarStyle = yaml_scalar_style_t(dec.event.style).toScalarStyleKind()
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
	case yaml_COMMENT_EVENT:
		evt.Kind = CommentEvent
		evt.Value = reflect.ValueOf(string(dec.event.value))
	case yaml_DOCUMENT_END_EVENT:
		evt.Kind = DocumentEndEvent
	case yaml_DOCUMENT_START_EVENT:
		evt.Kind = DocumentStartEvent
		// this isn't particularly useful now (we only accept 1.1, but it may be in the future)
		if dec.event.version_directive != nil {
			evt.YAMLVersion = &VersionInfo{dec.event.version_directive.major, dec.event.version_directive.minor}
		}
		// These are interesting, even if the parser automatically converts to the full form
		if dec.event.tag_directives != nil {
			evt.TagDefinitions = make([]TagInfo, len(dec.event.tag_directives))
			for i, dir := range dec.event.tag_directives {
				evt.TagDefinitions[i] = TagInfo{string(dir.handle), string(dir.prefix)}
			}
		}
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
