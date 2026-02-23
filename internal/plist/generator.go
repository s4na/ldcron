// Package plist provides launchd plist XML generation for ldcron jobs.
package plist

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/s4na/ldcron/internal/cron"
)

const scheduleKey = "X-Ldcron-Schedule"

// Generate creates the plist XML bytes for the given job parameters.
func Generate(label, schedule string, args []string, logDir string) ([]byte, error) {
	entries, err := cron.ParseSchedule(schedule)
	if err != nil {
		return nil, fmt.Errorf("cron式のパースに失敗: %w", err)
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
		return "", fmt.Errorf("LaunchAgentsディレクトリの作成に失敗: %w", err)
	}
	path := filepath.Join(dir, label+".plist")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("plistの書き込みに失敗: %w", err)
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
		return "", "", nil, err
	}
	if label == "" {
		base := filepath.Base(path)
		label = strings.TrimSuffix(base, ".plist")
	}
	return label, schedule, args, nil
}

// parsePlistInfoFromXML reads Label, X-Ldcron-Schedule, Program, and
// ProgramArguments from raw plist XML without requiring X-Ldcron-Schedule to
// be present. If ProgramArguments is absent but Program is set, args is
// returned as []string{program} to support both launchd plist variants.
func parsePlistInfoFromXML(data []byte) (label, schedule string, args []string, err error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var lastKey string
	var program string
	for {
		tok, xmlErr := dec.Token()
		if xmlErr != nil {
			if xmlErr != io.EOF {
				err = fmt.Errorf("XMLのデコードに失敗: %w", xmlErr)
			}
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "key":
				var s string
				if e := dec.DecodeElement(&s, &t); e == nil {
					lastKey = s
				}
			case "string":
				var s string
				if e := dec.DecodeElement(&s, &t); e == nil {
					switch lastKey {
					case "Label":
						label = s
					case scheduleKey:
						schedule = s
					case "Program":
						program = s
					}
				}
			case "array":
				if lastKey == "ProgramArguments" {
					args = decodeStringArray(dec, t)
				}
			}
		}
	}
	if len(args) == 0 && program != "" {
		args = []string{program}
	}
	return
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


func decodeStringArray(dec *xml.Decoder, _ xml.StartElement) []string {
	var result []string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "string" {
				var s string
				if e := dec.DecodeElement(&s, &t); e == nil {
					result = append(result, s)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "array" {
				return result
			}
		}
	}
	return result
}

