package export

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/dao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestTableData() *TableData {
	return &TableData{
		Title:      "TestTable",
		Headers:    []string{"ID", "Name", "State"},
		Rows:       [][]string{{"1", "job-one", "RUNNING"}, {"2", "job-two", "PENDING"}},
		ExportedAt: time.Now(),
	}
}

func TestTableExporterText(t *testing.T) {
	dir := t.TempDir()
	exp := NewTableExporter(dir)
	result, err := exp.Export(makeTestTableData(), FormatText, "")
	require.NoError(t, err)
	assert.True(t, result.Success)
	content, err := os.ReadFile(result.FilePath)
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "ID")
	assert.Contains(t, s, "job-one")
	assert.Contains(t, s, "RUNNING")
}

func TestTableExporterCSV(t *testing.T) {
	dir := t.TempDir()
	exp := NewTableExporter(dir)
	result, err := exp.Export(makeTestTableData(), FormatCSV, "")
	require.NoError(t, err)
	assert.True(t, result.Success)
	content, err := os.ReadFile(result.FilePath)
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "ID,Name,State")
	assert.Contains(t, s, "job-one")
}

func TestTableExporterJSON(t *testing.T) {
	dir := t.TempDir()
	exp := NewTableExporter(dir)
	result, err := exp.Export(makeTestTableData(), FormatJSON, "")
	require.NoError(t, err)
	assert.True(t, result.Success)
	raw, err := os.ReadFile(result.FilePath)
	require.NoError(t, err)

	var envelope struct {
		Title   string              `json:"title"`
		Total   int                 `json:"total"`
		Records []map[string]string `json:"records"`
	}
	require.NoError(t, json.Unmarshal(raw, &envelope))
	assert.Equal(t, "TestTable", envelope.Title)
	assert.Equal(t, 2, envelope.Total)
	assert.Equal(t, "job-one", envelope.Records[0]["Name"])
}

func TestTableExporterMarkdown(t *testing.T) {
	dir := t.TempDir()
	exp := NewTableExporter(dir)
	result, err := exp.Export(makeTestTableData(), FormatMarkdown, "")
	require.NoError(t, err)
	assert.True(t, result.Success)
	content, err := os.ReadFile(result.FilePath)
	require.NoError(t, err)
	s := string(content)
	assert.True(t, strings.HasPrefix(s, "# TestTable Export"))
	assert.Contains(t, s, "| ID | Name | State |")
	assert.Contains(t, s, "| 1 | job-one | RUNNING |")
}

func TestTableExporterHTML(t *testing.T) {
	dir := t.TempDir()
	exp := NewTableExporter(dir)
	result, err := exp.Export(makeTestTableData(), FormatHTML, "")
	require.NoError(t, err)
	assert.True(t, result.Success)
	content, err := os.ReadFile(result.FilePath)
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "<th>ID</th>")
	assert.Contains(t, s, "<td>job-one</td>")
}

func TestJobsTableData(t *testing.T) {
	now := time.Now()
	jobs := []*dao.Job{
		{ID: "100", Name: "myjob", User: "alice", Account: "eng", State: "RUNNING",
			Partition: "gpu", NodeCount: 2, Priority: 100, SubmitTime: now},
	}
	td := JobsTableData(jobs)
	assert.Equal(t, "Jobs", td.Title)
	require.Len(t, td.Rows, 1)
	assert.Equal(t, "100", td.Rows[0][0])
	assert.Equal(t, "myjob", td.Rows[0][1])
	assert.Equal(t, "RUNNING", td.Rows[0][4])
}

func TestNodesTableData(t *testing.T) {
	nodes := []*dao.Node{
		{Name: "node01", State: "IDLE", Partitions: []string{"cpu", "gpu"},
			CPUsTotal: 48, CPUsAllocated: 12, CPUsIdle: 36, MemoryTotal: 256000},
	}
	td := NodesTableData(nodes)
	assert.Equal(t, "Nodes", td.Title)
	require.Len(t, td.Rows, 1)
	assert.Equal(t, "node01", td.Rows[0][0])
	assert.Equal(t, "cpu,gpu", td.Rows[0][2])
	assert.Equal(t, "48", td.Rows[0][3])
}

func TestPartitionsTableData(t *testing.T) {
	parts := []*dao.Partition{
		{Name: "gpu", State: "UP", TotalNodes: 10, TotalCPUs: 480,
			DefaultTime: "1:00:00", MaxTime: "24:00:00"},
	}
	td := PartitionsTableData(parts)
	assert.Equal(t, "Partitions", td.Title)
	require.Len(t, td.Rows, 1)
	assert.Equal(t, "gpu", td.Rows[0][0])
}

func TestReservationsTableData(t *testing.T) {
	start := time.Now()
	end := start.Add(2 * time.Hour)
	reservations := []*dao.Reservation{
		{Name: "maint", State: "ACTIVE", StartTime: start, EndTime: end,
			Duration: 2 * time.Hour, NodeCount: 5},
	}
	td := ReservationsTableData(reservations)
	assert.Equal(t, "Reservations", td.Title)
	require.Len(t, td.Rows, 1)
	assert.Equal(t, "maint", td.Rows[0][0])
	assert.Equal(t, "5", td.Rows[0][6])
}

func TestQoSTableData(t *testing.T) {
	qosList := []*dao.QoS{
		{Name: "normal", Priority: 1000, MaxJobsPerUser: 50},
	}
	td := QoSTableData(qosList)
	assert.Equal(t, "QoS", td.Title)
	require.Len(t, td.Rows, 1)
	assert.Equal(t, "normal", td.Rows[0][0])
	assert.Equal(t, "1000", td.Rows[0][1])
}

func TestAccountsTableData(t *testing.T) {
	accounts := []*dao.Account{
		{Name: "research", Description: "Research group", Organization: "Uni",
			DefaultQoS: "normal"},
	}
	td := AccountsTableData(accounts)
	assert.Equal(t, "Accounts", td.Title)
	require.Len(t, td.Rows, 1)
	assert.Equal(t, "research", td.Rows[0][0])
}

func TestUsersTableData(t *testing.T) {
	users := []*dao.User{
		{Name: "alice", UID: 1001, DefaultAccount: "research",
			Accounts: []string{"research", "ml"}, AdminLevel: "None"},
	}
	td := UsersTableData(users)
	assert.Equal(t, "Users", td.Title)
	require.Len(t, td.Rows, 1)
	assert.Equal(t, "alice", td.Rows[0][0])
	assert.Equal(t, "1001", td.Rows[0][1])
	assert.Equal(t, "research,ml", td.Rows[0][3])
}
