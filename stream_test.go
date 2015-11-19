package yaml_test

import (
	"fmt"

	. "gopkg.in/check.v1"
	yaml "gopkg.in/yaml.v2"
)

var extraCommentRoundTripTests = []struct{
	in string
	out string
}{
	// before mapping key
	{
		"# this is important:\n# the following is a key\nkey: value\n",
		"# this is important:\n# the following is a key\nkey: value\n",
	},

	// between key and value
	{
		"key: # <-- this is a key\n  value\n",
		"key:\n  # <-- this is a key\n  value\n",
	},

	// multiline after key
	{
		"key: # <-- this is a key\n     # (no really, it is)\n  value\n",
		"key:\n  # <-- this is a key\n  # (no really, it is)\n  value\n",
	},

	// single line after block scalar marker
	{
		"key: | # <-- this is a block scalar\n  value\n  value\n  value\n",
		"key: | # <-- this is a block scalar\n  value\n  value\n  value\n",
	},

	// between key and submap
	{
		"key1: # <-- this is a key\n  key2: value1\n  key3: value2\n",
		"key1:\n  # <-- this is a key\n  key2: value1\n  key3: value2\n",
	},

	// with bare sequences
	{
		"# this is a slice\n- item 1\n# another item\n- item 2\n",
		"# this is a slice\n- item 1\n# another item\n- item 2\n",
	},

	// with sequence in map
	{
		"key:\n# this is a slice\n- item 1\n# another item\n- item 2\n",
		"key:\n# this is a slice\n- item 1\n# another item\n- item 2\n",
	},

	// with map in sequence
	{
		"- # this is actually a mapping\n  key: value\n# here's another mapping\n- key: value\n",
		"# this is actually a mapping\n- key: value\n# here's another mapping\n- key: value\n",
	},

	// bare comment only
	{
		"# this document is empty",
		"# this document is empty\nnull\n",
	},

	// comment before scalar
	{
		"# comment\nfoo\n",
		"# comment\nfoo\n",
	},

	// comment in empty doc
	{
		"---\n# empty\n...",
		"---\n# empty\nnull\n...\n",
	},

}

func encodeEvents(decoder *yaml.StreamDecoder) (string, error) {
	encoder := yaml.NewExplicitStreamEncoder()
	for evt := decoder.NextEvent(); evt.Kind != yaml.FinishEvent; evt = decoder.NextEvent() {
		tag := evt.Tag
		if evt.Implicit || evt.QuotedImplicit {
			tag = ""
		}

		switch evt.Kind {
		case yaml.ScalarEvent:
			if tag == "tag:yaml.org,2002:binary" {
				encoder.EmitValue("", evt.Value, evt.Flow)
			} else {
				encoder.EmitValue(tag, evt.Value, evt.Flow)
			}
		case yaml.MappingStartEvent:
			encoder.BeginMapping("", tag, evt.Flow)
		case yaml.MappingEndEvent:
			encoder.EndMapping()
		case yaml.SequenceStartEvent:
			encoder.BeginSequence("", tag, evt.Flow)
		case yaml.SequenceEndEvent:
			encoder.EndSequence()
		case yaml.CommentEvent:
			encoder.EmitComment(evt.Value.String(), true)
		case yaml.DocumentStartEvent:
			encoder.BeginDocument(evt.YAMLVersion, evt.TagDefinitions, evt.Implicit)
		case yaml.DocumentEndEvent:
			encoder.EndDocument(evt.Implicit)
		default:
			return "", fmt.Errorf("Unknown event: %+v", evt)
		}
	}

	return string(encoder.Finish()), nil
}


func (s *S) TestRoundTrip(c *C) {
	for _, item := range marshalTests {
		decoder := yaml.NewStreamDecoder([]byte(item.data))
		newYaml, err := encodeEvents(decoder)
		c.Assert(err, IsNil)
		c.Assert(newYaml, Equals, item.data)
	}

	for _, item := range extraCommentRoundTripTests {
		decoder := yaml.NewStreamDecoder([]byte(item.in))
		newYaml, err := encodeEvents(decoder)
		c.Assert(err, IsNil)
		c.Assert(newYaml, Equals, item.out)
	}
}
