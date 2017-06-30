package presenters

import (
	"crucible/models"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
)

func PrintJobs(jobs []models.Job, stdout io.Writer) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 1, ' ', 0)

	printRow(tw, "Name", "Pid", "Status")
	for _, job := range jobs {
		printRow(tw, job.Name, strconv.Itoa(job.Pid), job.Status)
	}

	return tw.Flush()
}

func printRow(w io.Writer, args ...string) {
	row := strings.Join(args, "\t")
	fmt.Fprintf(w, "%s\n", row)
}
