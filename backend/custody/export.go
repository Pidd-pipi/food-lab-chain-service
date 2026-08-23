package custody

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// ToCSV renders the chains as a CSV document with a header row.
func ToCSV(chains []Chain) (string, error) {
	var builder strings.Builder
	writer := bufio.NewWriter(&builder)
	if _, err := writer.WriteString("specimen,step,event,actor,action,at\n"); err != nil {
		return "", err
	}
	for _, chain := range chains {
		for _, node := range chain.Nodes {
			if _, err := fmt.Fprintf(writer, "%s,%d,%s,%s,%s,%s\n",
				chain.SpecimenID, node.Step, node.EventID, node.Actor, node.Action, node.At); err != nil {
				return "", err
			}
		}
	}
	if err := writer.Flush(); err != nil {
		return "", err
	}
	return builder.String(), nil
}

// ExportAll streams the chains to w in tab-separated form and returns the
// number of rows written. The writer is flushed exactly once at the end; an
// earlier write or context error is never swallowed by a successful flush, and a
// flush failure is reported only when no prior error occurred.
func ExportAll(ctx context.Context, w io.Writer, chains []Chain) (written int, err error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	writer := bufio.NewWriter(w)
	defer func() {
		// Preserve any error already observed during the write loop. Only when
		// the loop ran clean do we surface a flush failure, so a successful
		// flush can never mask a real error or make a truncated export look
		// complete.
		if err != nil {
			_ = writer.Flush()
			return
		}
		err = writer.Flush()
	}()
	for _, chain := range chains {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		for _, node := range chain.Nodes {
			if _, err := fmt.Fprintf(writer, "%s\t%d\n", node.EventID, node.Step); err != nil {
				return written, err
			}
			written++
		}
	}
	return written, nil
}
