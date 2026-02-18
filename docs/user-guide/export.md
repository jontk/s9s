# Export Guide

Export data from any s9s view to a file for analysis, reporting, or integration with other tools.

## Quick Start

In any supported view, press **`e`** to open the export dialog:

1. Select a **format** (CSV is the default)
2. Confirm or change the **output directory** (`~/slurm_exports` by default)
3. Click **Export**

A confirmation modal shows the full path of the saved file when done.

## Supported Views

Export is available in all main data views:

| View | Key | Exported Data |
|------|-----|---------------|
| Jobs | `e` | ID, Name, User, Account, State, Partition, Nodes, Time Used, Time Limit, Priority, Submit Time |
| Nodes | `e` | Name, State, Partitions, CPU totals/allocated/idle/load, Memory totals, Features, Reason |
| Partitions | `e` | Name, State, Total Nodes, Total CPUs, Default Time, Max Time, QoS, Nodes |
| Reservations | `e` | Name, State, Start/End Time, Duration, Nodes, Node Count, Core Count, Users, Accounts |
| QoS | `e` | Name, Priority, Preempt Mode, Flags, all per-user/account limits |
| Accounts | `e` | Name, Description, Organization, Parent, QoS, Coordinators, resource limits |
| Users | `e` | Name, UID, Default Account, Accounts, Admin Level, QoS, resource limits |

## Supported Formats

| Format | Extension | Best for |
|--------|-----------|----------|
| **CSV** | `.csv` | Spreadsheets, data processing (default) |
| **JSON** | `.json` | Programmatic processing, API integration |
| **Text** | `.txt` | Human-readable ASCII tables, logs |
| **Markdown** | `.md` | Documentation, GitHub reports |
| **HTML** | `.html` | Sharing, browser viewing |

## Export Dialog

```
┌─────────────── Export Jobs (42 records) ───────────────┐
│                                                         │
│  Format:   [CSV           ▼]                           │
│                                                         │
│  Save to:  ~/slurm_exports                             │
│                                                         │
│  [ Export ]  [ Cancel ]                                 │
│                                                         │
└─────────────────────────────────────────────────────────┘
  [Tab] Navigate  [Enter] Select  [Esc] Cancel
```

- **Format** — dropdown with 5 options; press Enter to open
- **Save to** — directory path; files are named automatically with a timestamp (e.g. `jobs_20260218_143022.csv`)
- **Export** — writes the file and shows a result modal
- **Cancel / Esc** — closes the dialog without writing anything

## Output Files

Files are written to the specified directory with timestamped names:

```
~/slurm_exports/
  jobs_20260218_143022.csv
  nodes_20260218_144501.json
  partitions_20260218_150012.md
```

The directory is created automatically if it doesn't exist.

## Format Examples

### CSV

```
ID,Name,User,Account,State,Partition,Nodes,Time Used,Time Limit,Priority,Submit Time
1234,train-bert,alice,ml,RUNNING,gpu,4,01:23:45,24:00:00,100,2026-02-18 12:00:00
1235,preprocess,bob,eng,PENDING,cpu,1,,04:00:00,50,2026-02-18 12:05:00
```

### JSON

```json
{
  "title": "Jobs",
  "exported_at": "2026-02-18T14:30:22Z",
  "total": 2,
  "records": [
    {"ID": "1234", "Name": "train-bert", "State": "RUNNING", ...},
    {"ID": "1235", "Name": "preprocess", "State": "PENDING", ...}
  ]
}
```

### Markdown

```markdown
# Jobs Export

_Exported at: 2026-02-18 14:30:22 — 2 records_

| ID | Name | User | State | ...  |
| --- | --- | --- | --- | --- |
| 1234 | train-bert | alice | RUNNING | ... |
```

### Text

```
Jobs Export
Exported at: 2026-02-18 14:30:22
Total records: 2

+------+------------+-------+---------+
| ID   | Name       | User  | State   |
+------+------------+-------+---------+
| 1234 | train-bert | alice | RUNNING |
| 1235 | preprocess | bob   | PENDING |
+------+------------+-------+---------+
```

## Tips

- **Filters are respected** — only the data currently loaded into the view is exported. Apply filters before exporting to narrow results.
- **The data is a point-in-time snapshot** — exported at the moment you press Export.
- **Path security** — export paths are validated; only paths within your home directory are allowed.

## See Also

- [Filtering](./filtering.md) — narrow data before exporting
- [Batch Operations](./batch-operations.md) — act on multiple jobs at once
- [Keyboard Shortcuts](./keyboard-shortcuts.md) — full key reference
