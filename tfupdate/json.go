package tfupdate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// jsonValue retains member order, repeated block names, and original byte ranges.
type jsonValue struct {
	value   any
	start   int
	end     int
	members []jsonMember
	items   []*jsonValue
}

type jsonMember struct {
	name  string
	value *jsonValue
}

type jsonEdit struct {
	start, end int
	text       []byte
}

type jsonDocument struct {
	input []byte
	edits []jsonEdit
}

// readJSONValue is called only after json.Valid has checked the entire input.
// Decoder supplies decoded strings without losing the original value ranges.
func readJSONValue(d *json.Decoder, input []byte) *jsonValue {
	start := int(d.InputOffset())
	for strings.ContainsRune(" \t\r\n,:", rune(input[start])) {
		start++
	}
	token, _ := d.Token()
	n := &jsonValue{value: token, start: start}
	switch token {
	case json.Delim('{'):
		for d.More() {
			key, _ := d.Token()
			n.members = append(n.members, jsonMember{key.(string), readJSONValue(d, input)})
		}
		_, _ = d.Token()
	case json.Delim('['):
		for d.More() {
			n.items = append(n.items, readJSONValue(d, input))
		}
		_, _ = d.Token()
	}
	n.end = int(d.InputOffset())
	return n
}

// objects interprets one block or label collection, not arbitrary expressions.
func (n *jsonValue) objects() ([]*jsonValue, error) {
	switch n.value {
	case nil:
		return nil, nil
	case json.Delim('{'):
		return []*jsonValue{n}, nil
	case json.Delim('['):
		for _, item := range n.items {
			if item.value != json.Delim('{') {
				return nil, fmt.Errorf("expected a block object at byte %d", item.start)
			}
		}
		return n.items, nil
	default:
		return nil, fmt.Errorf("expected a block object or array at byte %d", n.start)
	}
}

func (n *jsonValue) argument(name string) (*jsonValue, error) {
	var found *jsonValue
	for _, member := range n.members {
		if member.name != name {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("duplicate argument %q at byte %d", name, member.value.start)
		}
		found = member.value
	}
	return found, nil
}

func (n *jsonValue) stringValue() (string, error) {
	v, ok := n.value.(string)
	if !ok {
		return "", fmt.Errorf("expected a string at byte %d", n.start)
	}
	return v, nil
}

func (n *jsonValue) blocks(name string, labeled bool, visit func(string, *jsonValue) error) error {
	for _, member := range n.members {
		if member.name != name {
			continue
		}
		objects, err := member.value.objects()
		if err != nil {
			return err
		}
		for _, object := range objects {
			if !labeled {
				if err := visit("", object); err != nil {
					return err
				}
				continue
			}
			for _, label := range object.members {
				bodies, err := label.value.objects()
				if err != nil {
					return err
				}
				for _, body := range bodies {
					if err := visit(label.name, body); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (d *jsonDocument) replace(n *jsonValue, value string) error {
	old, err := n.stringValue()
	if err != nil {
		return err
	}
	if old != value {
		encoded, _ := json.Marshal(value)
		d.edits = append(d.edits, jsonEdit{n.start, n.end, encoded})
	}
	return nil
}

func (d *jsonDocument) version(body *jsonValue, name, value string) error {
	n, err := body.argument(name)
	if err != nil {
		return err
	}
	if n == nil {
		return nil
	}
	return d.replace(n, value)
}

func (d *jsonDocument) providers(root *jsonValue, mc *ModuleContext, o Option) error {
	name := o.name
	if strings.Contains(name, "/") {
		name = mc.ResolveProviderShortNameFromSource(name)
	}
	err := root.blocks("terraform", false, func(_ string, body *jsonValue) error {
		return body.blocks("required_providers", false, func(_ string, providers *jsonValue) error {
			seen := make(map[string]bool)
			for _, member := range providers.members {
				if member.name == "//" {
					continue
				}
				if seen[member.name] {
					return fmt.Errorf("duplicate provider argument %q", member.name)
				}
				seen[member.name] = true
				p := member.value
				version := p
				if p.value == json.Delim('{') {
					if _, err := p.argument("source"); err != nil {
						return err
					}
					var err error
					version, err = p.argument("version")
					if err != nil {
						return err
					}
				}
				if name == "" || member.name != name || version == nil {
					continue
				}
				if err := d.replace(version, o.version); err != nil {
					return err
				}
			}
			return nil
		})
	})
	if err != nil {
		return err
	}
	return root.blocks("provider", true, func(label string, body *jsonValue) error {
		version, err := body.argument("version")
		if err != nil {
			return err
		}
		if label != o.name || version == nil {
			return nil
		}
		return d.replace(version, o.version)
	})
}

func (d *jsonDocument) modules(root *jsonValue, o Option) error {
	return root.blocks("module", true, func(_ string, body *jsonValue) error {
		source, err := body.argument("source")
		if err != nil {
			return err
		}
		version, err := body.argument("version")
		if err != nil {
			return err
		}
		if source == nil {
			return nil
		}
		name, err := source.stringValue()
		if err != nil {
			return err
		}
		matched := moduleSourceRegexp.FindStringSubmatch(name)
		if matched != nil {
			name = matched[1]
		}
		u := ModuleUpdater{name: o.name, nameRegex: o.nameRegex}
		if !u.match(name) {
			return nil
		}
		if matched != nil {
			return d.replace(source, name+"?ref=v"+o.version)
		}
		if version == nil {
			return nil
		}
		return d.replace(version, o.version)
	})
}

func updateJSON(mc *ModuleContext, input []byte, w io.Writer, filename string) (bool, error) {
	if !json.Valid(input) {
		return false, fmt.Errorf("failed to parse JSON input: %s", filename)
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	root := readJSONValue(decoder, input)
	if root.value != json.Delim('{') {
		return false, fmt.Errorf("expected a JSON object: %s", filename)
	}
	d := jsonDocument{input: input}
	o := mc.Option()
	var err error
	switch o.updateType {
	case "terraform", "opentofu":
		err = root.blocks("terraform", false, func(_ string, body *jsonValue) error {
			return d.version(body, "required_version", o.version)
		})
	case "provider":
		err = d.providers(root, mc, o)
	case "module":
		err = d.modules(root, o)
	}
	if err != nil {
		return false, fmt.Errorf("failed to update JSON input %s: %w", filename, err)
	}

	// Selection order can differ from source order (provider and terraform blocks).
	sort.Slice(d.edits, func(i, j int) bool { return d.edits[i].start < d.edits[j].start })
	var output bytes.Buffer
	pos := 0
	for _, edit := range d.edits {
		output.Write(d.input[pos:edit.start])
		output.Write(edit.text)
		pos = edit.end
	}
	output.Write(d.input[pos:])
	n, err := w.Write(output.Bytes())
	if err == nil && n != output.Len() {
		err = io.ErrShortWrite
	}
	if err != nil {
		return false, fmt.Errorf("failed to write JSON output %s: %w", filename, err)
	}
	return len(d.edits) > 0, nil
}
