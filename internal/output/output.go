package output

import (
	"encoding/json"
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

const (
	FormatTable = "table"
	FormatJSON  = "json"
)

func AddFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("output", "o", FormatTable, `Output format: "table" or "json"`)
}

func Format(cmd *cobra.Command) string {
	f, _ := cmd.Flags().GetString("output")
	return f
}

func JSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func Table(w io.Writer, headers []string, rows [][]string) error {
	t := tablewriter.NewTable(w,
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignLeft}},
			Row:    tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignLeft}},
		}),
	)
	t.Header(headers)
	for _, row := range rows {
		if err := t.Append(row); err != nil {
			return err
		}
	}
	return t.Render()
}
