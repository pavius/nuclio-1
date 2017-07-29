package render

import (
	"github.com/olekukonko/tablewriter"
	"io"
	"github.com/nuclio/nuclio/pkg/functioncr"
	"github.com/ghodss/yaml"
	"fmt"
	"bytes"
	"encoding/json"
	"strings"
)

// return string list based on resource type
func Obj2Strings(obj interface{}, wide bool) []string {
	switch obj.(type) {
	case *functioncr.Function:
		fc := obj.(*functioncr.Function)
		port := fmt.Sprintf("%d",fc.Spec.HTTPPort)
		return []string{fc.Namespace,fc.Labels["function"],fc.Labels["version"],string(fc.Status.State), port,fc.Spec.Image}
	default:
		return []string{}
	}
}


type TablePrinter struct {
	writer io.Writer
	head []string
}

func NewTablePrinter(writer io.Writer, head []string) *TablePrinter {
	return &TablePrinter{writer:writer, head:head}
}

func (tp *TablePrinter) Print(data *[][]string) {

	table := tablewriter.NewWriter(tp.writer)
	table.SetHeader(tp.head)
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetHeaderLine(false)
	table.AppendBulk(*data) // Add Bulk Data
	table.Render()
}

func PrintInterface(writer io.Writer, item interface{}, format string) error {
	switch format {
	case "json":
		return Print_json(writer, item)
	case "yaml":
		return Print_yaml(writer, item)
	default:
		return nil
	}
}

func Print_json(writer io.Writer, item interface{}) error {
	body, err := json.Marshal(item)
	if err != nil { return err}

	var pbody bytes.Buffer
	err = json.Indent(&pbody, body, "", "\t")
	if err != nil { return err}

	fmt.Fprintln(writer, string(pbody.Bytes()))
	return nil
}

func Print_yaml(writer io.Writer, item interface{}) error {
	body, err := yaml.Marshal(item)
	if err != nil { return err}
	fmt.Fprintln(writer, string(body))
	return nil
}

func Map2Str( in map[string]string) string {
	list := []string{}
	for k, v := range in {
		list = append(list, k + "=" + v)
	}
	return strings.Join(list, ",")
}