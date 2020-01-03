package table

import (
	"bytes"
	"strings"
)

func SimpleColumnFormat(value string) string {
	if strings.Contains(value, "{{") {
		return value
	}
	return "{{." + value + "}}"
}

func SimpleFormat(values [][]string) (string, string) {
	headerBuffer := bytes.Buffer{}
	valueBuffer := bytes.Buffer{}
	for _, v := range values {
		appendTabDelim(&headerBuffer, v[0])
		v1 := SimpleColumnFormat(v[1])
		if v1 != "" {
			appendTabDelim(&valueBuffer, v1)
		}
	}

	headerBuffer.WriteString("\n")
	valueBuffer.WriteString("\n")

	return headerBuffer.String(), valueBuffer.String()
}

func appendTabDelim(buf *bytes.Buffer, value string) {
	if buf.Len() == 0 {
		buf.WriteString(value)
	} else {
		buf.WriteString("\t")
		buf.WriteString(value)
	}
}
