package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
)

func writeJsonLine(out *bufio.Writer, columns []string, values []any) {
	out.WriteByte('{')

	for i, col := range columns {
		val := values[i]

		// _cursor is a special case
		if i == 0 {
			val := values[i]
			fmt.Fprintf(out, `"%s":"%016x"`, col, val)
			continue
		}

		out.WriteByte(',')
		out.WriteByte('"')
		out.WriteString(col)
		out.WriteByte('"')
		out.WriteByte(':')

		if v, ok := val.([]byte); ok {
			val = string(v)
		}

		s, _ := json.Marshal(val)
		out.Write(s)
	}

	out.WriteByte('}')
	out.WriteByte('\n')

	out.Flush()
}
