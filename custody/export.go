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
// number of rows written. The writer is flushed exactly once at the end; a
// flush failure is reported on the success path, and an earlier write error is
// never swallowed by the flush.
func ExportAll(ctx context.Context, w io.Writer, chains []Chain) (written int, err error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	writer := bufio.NewWriter(w)
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
	if err := writer.Flush(); err != nil {
		return written, err
	}
	return written, nil
}
