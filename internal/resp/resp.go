package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ReadCommand reads a RESP array of bulk strings. It also accepts simple
// inline commands (for example: PING) to make manual testing easier.
func ReadCommand(r *bufio.Reader) ([]string, error) {
	first, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	if first != '*' {
		rest, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		line := strings.TrimSpace(string(first) + rest)
		if line == "" {
			return nil, fmt.Errorf("empty command")
		}
		return strings.Fields(line), nil
	}

	countLine, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	count, err := strconv.Atoi(strings.TrimSpace(countLine))
	if err != nil || count < 1 || count > 1024 {
		return nil, fmt.Errorf("invalid RESP array length")
	}

	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		prefix, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if prefix != '$' {
			return nil, fmt.Errorf("expected bulk string")
		}
		lengthLine, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(strings.TrimSpace(lengthLine))
		if err != nil || length < 0 || length > 64*1024*1024 {
			return nil, fmt.Errorf("invalid bulk string length")
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		if buf[length] != '\r' || buf[length+1] != '\n' {
			return nil, fmt.Errorf("invalid bulk string terminator")
		}
		args = append(args, string(buf[:length]))
	}
	return args, nil
}

func Simple(s string) []byte { return []byte("+" + s + "\r\n") }
func Error(s string) []byte  { return []byte("-ERR " + clean(s) + "\r\n") }
func Bulk(s string) []byte   { return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)) }
func NullBulk() []byte       { return []byte("$-1\r\n") }
func Integer(n int64) []byte { return []byte(fmt.Sprintf(":%d\r\n", n)) }

func Array(items []string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "*%d\r\n", len(items))
	for _, item := range items {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(item), item)
	}
	return []byte(b.String())
}

func clean(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r", " "), "\n", " ")
}
