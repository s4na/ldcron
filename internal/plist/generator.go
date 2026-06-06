// Package plist provides launchd plist XML generation for ldcron jobs.
package plist

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/s4na/ldcron/internal/cron"
)

const scheduleKey = "X-Ldcron-Schedule"

// Generate creates the plist XML bytes for the given job parameters.
func Generate(label, schedule string, args []string, logDir string) ([]byte, error) {
	entries, err := cron.ParseSchedule(schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cron expression: %w", err)
	}

	// Extract ID from label: com.ldcron.<id>
	id := strings.TrimPrefix(label, "com.ldcron.")
	if id == "" {
		id = label
	}
	logPath := filepath.Join(logDir, id+".log")

	const header = `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n"

	doc := buildDocument(label, schedule, args, entries, logPath)
	body, err := xml.MarshalIndent(doc, "", "\t")
	if err != nil {
		return nil, err
	}
	buf := []byte(header)
	buf = append(buf, body...)
	buf = append(buf, '\n')
	return buf, nil
}

// Write writes the plist file to dir/<label>.plist and returns the path.
func Write(dir, label, schedule string, args []string, logDir string) (string, error) {
	data, err := Generate(label, schedule, args, logDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}
	path := filepath.Join(dir, label+".plist")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write plist: %w", err)
	}
	return path, nil
}

// ReadPlistInfo reads Label, X-Ldcron-Schedule (optional), and ProgramArguments
// from any launchd plist file. If X-Ldcron-Schedule is absent, schedule is
// returned as an empty string without error. If Label is absent in the plist,
// the filename stem is used as the label.
func ReadPlistInfo(path string) (label, schedule string, args []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil, err
	}
	label, schedule, args, err = parsePlistInfoFromXML(data)
	if err != nil {
		normalized, normalizeErr := normalizePlistXML(path)
		if normalizeErr != nil {
			return "", "", nil, fmt.Errorf("%w; failed to normalize plist with plutil: %v", err, normalizeErr)
		}
		label, schedule, args, err = parsePlistInfoFromXML(normalized)
		if err != nil {
			return "", "", nil, err
		}
	}
	if label == "" {
		base := filepath.Base(path)
		label = strings.TrimSuffix(base, ".plist")
	}
	return label, schedule, args, nil
}

// normalizePlistXML asks macOS' native plist tool to render any valid plist
// representation (including binary plists) as XML.
func normalizePlistXML(path string) ([]byte, error) {
	cmd := exec.Command("/usr/bin/plutil", "-convert", "xml1", "-o", "-", "--", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// parsePlistInfoFromXML reads top-level Label, X-Ldcron-Schedule, Program,
// BundleProgram, and ProgramArguments from raw plist XML without requiring
// X-Ldcron-Schedule to be present.
func parsePlistInfoFromXML(data []byte) (label, schedule string, args []string, err error) {
	dict, err := decodeRootDict(data)
	if err != nil {
		return "", "", nil, err
	}

	label = stringValue(dict["Label"])
	schedule = stringValue(dict[scheduleKey])
	program := stringValue(dict["Program"])
	bundleProgram := stringValue(dict["BundleProgram"])
	programArgs := stringArrayValue(dict["ProgramArguments"])
	args = launchdCommandArgs(program, bundleProgram, programArgs)
	return label, schedule, args, nil
}

type plistValue struct {
	kind  string
	str   string
	array []string
}

func decodeRootDict(data []byte) (map[string]plistValue, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, xmlErr := dec.Token()
		if xmlErr != nil {
			if xmlErr != io.EOF {
				return nil, fmt.Errorf("failed to decode XML: %w", xmlErr)
			}
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == "plist" {
			continue
		}
		if start.Name.Local != "dict" {
			return nil, fmt.Errorf("plist root must contain a dict, got <%s>", start.Name.Local)
		}
		return decodePlistDict(dec)
	}
	return nil, fmt.Errorf("plist root dict not found")
}

func decodePlistValue(dec *xml.Decoder, start xml.StartElement) (plistValue, error) {
	switch start.Name.Local {
	case "dict":
		if _, err := decodePlistDict(dec); err != nil {
			return plistValue{}, err
		}
		return plistValue{kind: "dict"}, nil
	case "array":
		return decodePlistArray(dec)
	case "string", "integer", "real", "date", "data":
		var s string
		if err := dec.DecodeElement(&s, &start); err != nil {
			return plistValue{}, fmt.Errorf("failed to decode <%s>: %w", start.Name.Local, err)
		}
		return plistValue{kind: start.Name.Local, str: s}, nil
	case "true", "false":
		var discard struct{}
		if err := dec.DecodeElement(&discard, &start); err != nil {
			return plistValue{}, fmt.Errorf("failed to decode <%s>: %w", start.Name.Local, err)
		}
		return plistValue{kind: "bool", str: start.Name.Local}, nil
	default:
		var discard struct{}
		if err := dec.DecodeElement(&discard, &start); err != nil {
			return plistValue{}, fmt.Errorf("failed to skip <%s>: %w", start.Name.Local, err)
		}
		return plistValue{kind: start.Name.Local}, nil
	}
}

func decodePlistDict(dec *xml.Decoder) (map[string]plistValue, error) {
	result := make(map[string]plistValue)
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to decode dict: %w", err)
		}
		switch t := tok.(type) {
		case xml.EndElement:
			if t.Name.Local == "dict" {
				return result, nil
			}
		case xml.StartElement:
			if t.Name.Local != "key" {
				return nil, fmt.Errorf("expected <key> in dict, got <%s>", t.Name.Local)
			}
			var key string
			if err := dec.DecodeElement(&key, &t); err != nil {
				return nil, fmt.Errorf("failed to decode dict key: %w", err)
			}
			valueStart, err := nextStartElement(dec)
			if err != nil {
				return nil, fmt.Errorf("missing value for key %q: %w", key, err)
			}
			value, err := decodePlistValue(dec, valueStart)
			if err != nil {
				return nil, fmt.Errorf("failed to decode value for key %q: %w", key, err)
			}
			result[key] = value
		}
	}
}

func decodePlistArray(dec *xml.Decoder) (plistValue, error) {
	var result []string
	for {
		tok, err := dec.Token()
		if err != nil {
			return plistValue{}, fmt.Errorf("failed to decode array: %w", err)
		}
		switch t := tok.(type) {
		case xml.EndElement:
			if t.Name.Local == "array" {
				return plistValue{kind: "array", array: result}, nil
			}
		case xml.StartElement:
			value, err := decodePlistValue(dec, t)
			if err != nil {
				return plistValue{}, err
			}
			if value.kind == "string" {
				result = append(result, value.str)
			}
		}
	}
}

func nextStartElement(dec *xml.Decoder) (xml.StartElement, error) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return xml.StartElement{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			return t, nil
		case xml.EndElement:
			return xml.StartElement{}, fmt.Errorf("unexpected </%s>", t.Name.Local)
		}
	}
}

func stringValue(value plistValue) string {
	if value.kind != "string" {
		return ""
	}
	return value.str
}

func stringArrayValue(value plistValue) []string {
	if value.kind != "array" {
		return nil
	}
	result := make([]string, len(value.array))
	copy(result, value.array)
	return result
}

func launchdCommandArgs(program, bundleProgram string, programArgs []string) []string {
	executable := program
	if executable == "" {
		executable = bundleProgram
	}
	if executable == "" {
		return programArgs
	}
	if len(programArgs) <= 1 {
		return []string{executable}
	}
	args := make([]string, 0, len(programArgs))
	args = append(args, executable)
	args = append(args, programArgs[1:]...)
	return args
}

// --- XML document model (hand-rolled to match Apple plist DTD) ---

type plistDoc struct {
	XMLName xml.Name `xml:"plist"`
	Version string   `xml:"version,attr"`
	Dict    dictNode
}

type dictNode struct {
	XMLName xml.Name  `xml:"dict"`
	Entries []xmlNode `xml:",any"`
}

type xmlNode struct {
	XMLName xml.Name
	Content string    `xml:",chardata"`
	Items   []xmlNode `xml:",any"`
}

func keyNode(name string) xmlNode {
	return xmlNode{XMLName: xml.Name{Local: "key"}, Content: name}
}

func strNode(val string) xmlNode {
	return xmlNode{XMLName: xml.Name{Local: "string"}, Content: val}
}

func intNode(val int) xmlNode {
	return xmlNode{XMLName: xml.Name{Local: "integer"}, Content: fmt.Sprintf("%d", val)}
}

func buildDocument(label, schedule string, args []string, entries []cron.CalendarEntry, logPath string) plistDoc {
	d := dictNode{}

	// Label
	d.Entries = append(d.Entries, keyNode("Label"), strNode(label))

	// ProgramArguments
	argItems := make([]xmlNode, len(args))
	for i, a := range args {
		argItems[i] = strNode(a)
	}
	d.Entries = append(d.Entries,
		keyNode("ProgramArguments"),
		xmlNode{XMLName: xml.Name{Local: "array"}, Items: argItems},
	)

	// StartCalendarInterval
	calItems := buildCalendarItems(entries)
	d.Entries = append(d.Entries,
		keyNode("StartCalendarInterval"),
		xmlNode{XMLName: xml.Name{Local: "array"}, Items: calItems},
	)

	// Log paths
	d.Entries = append(d.Entries,
		keyNode("StandardOutPath"), strNode(logPath),
		keyNode("StandardErrorPath"), strNode(logPath),
	)

	// Metadata: original cron expression
	d.Entries = append(d.Entries, keyNode(scheduleKey), strNode(schedule))

	return plistDoc{Version: "1.0", Dict: d}
}

func buildCalendarItems(entries []cron.CalendarEntry) []xmlNode {
	items := make([]xmlNode, 0, len(entries))
	for _, e := range entries {
		var kv []xmlNode
		if e.Minute != nil {
			kv = append(kv, keyNode("Minute"), intNode(*e.Minute))
		}
		if e.Hour != nil {
			kv = append(kv, keyNode("Hour"), intNode(*e.Hour))
		}
		if e.Day != nil {
			kv = append(kv, keyNode("Day"), intNode(*e.Day))
		}
		if e.Month != nil {
			kv = append(kv, keyNode("Month"), intNode(*e.Month))
		}
		if e.Weekday != nil {
			kv = append(kv, keyNode("Weekday"), intNode(*e.Weekday))
		}
		items = append(items, xmlNode{XMLName: xml.Name{Local: "dict"}, Items: kv})
	}
	return items
}
