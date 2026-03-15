package config

// JobSubmissionValues holds job submission field values parsed from config maps.
// Its fields mirror dao.JobSubmission so callers can map between them directly.
type JobSubmissionValues struct {
	Name        string
	Script      string
	Partition   string
	Account     string
	QoS         string
	Nodes       int
	CPUs        int
	Memory      string
	GPUs        int
	TimeLimit   string
	WorkingDir  string
	OutputFile  string
	ErrorFile   string
	EmailNotify bool
	Email       string
	ArraySpec     string
	Exclusive     bool
	Requeue       bool
	Constraints   string
	NTasks        int
	NTasksPerNode int
	Gres          string
	Hold          bool
	Reservation   string
	Licenses      string
	Wckey         string
	ExcludeNodes  string
	Priority      int
	Nice          int
	MemoryPerCPU       string
	BeginTime          string
	Comment            string
	Distribution       string
	Prefer             string
	RequiredNodes      string
	StandardInput      string
	Container          string
	ThreadsPerCore     int
	TasksPerCore       int
	TasksPerSocket     int
	SocketsPerNode     int
	MaximumNodes       int
	MaximumCPUs        int
	MinimumCPUsPerNode int
	TimeMinimum        string
	Contiguous         bool
	Overcommit         bool
	KillOnNodeFail     bool
	WaitAllNodes       bool
	OpenMode           string
	TRESPerTask         string
	TRESPerSocket       string
	Signal              string
	TmpDiskPerNode      int
	Deadline            string
	NTasksPerTRES       int
	CPUBinding          string
	CPUFrequency        string
	Network             string
	X11                 string
	Immediate           bool
	BurstBuffer         string
	BatchFeatures       string
	TRESBind            string
	TRESFreq            string
	CoreSpecification   int
	ThreadSpecification int
	MemoryBinding       string
	MinimumCPUs         int
	TRESPerJob          string
	CPUsPerTRES         string
	MemoryPerTRES       string
	Argv                string
	Flags               string
	ProfileTypes        string
	CPUBindingFlags     string
	MemoryBindingType   string
	RequiredSwitches    int
	WaitForSwitch       int
	ClusterConstraint   string
	Clusters            string
	Dependencies        []string
}

// JobSubmissionFromMap converts a map of camelCase config keys to JobSubmissionValues.
// It handles type assertions safely and supports both int and float64 for numeric fields,
// since YAML/JSON unmarshaling may produce either type.
func JobSubmissionFromMap(m map[string]any) JobSubmissionValues {
	var js JobSubmissionValues

	if v, ok := m["name"].(string); ok {
		js.Name = v
	}
	if v, ok := m["script"].(string); ok {
		js.Script = v
	}
	if v, ok := m["partition"].(string); ok {
		js.Partition = v
	}
	if v, ok := m["account"].(string); ok {
		js.Account = v
	}
	if v, ok := m["qos"].(string); ok {
		js.QoS = v
	}
	if v, ok := m["nodes"]; ok {
		js.Nodes = toInt(v)
	}
	if v, ok := m["cpus"]; ok {
		js.CPUs = toInt(v)
	}
	if v, ok := m["memory"].(string); ok {
		js.Memory = v
	}
	if v, ok := m["gpus"]; ok {
		js.GPUs = toInt(v)
	}
	if v, ok := m["timeLimit"].(string); ok {
		js.TimeLimit = v
	}
	if v, ok := m["workingDir"].(string); ok {
		js.WorkingDir = v
	}
	if v, ok := m["outputFile"].(string); ok {
		js.OutputFile = v
	}
	if v, ok := m["errorFile"].(string); ok {
		js.ErrorFile = v
	}
	if v, ok := m["emailNotify"].(bool); ok {
		js.EmailNotify = v
	}
	if v, ok := m["email"].(string); ok {
		js.Email = v
	}
	if v, ok := m["arraySpec"].(string); ok {
		js.ArraySpec = v
	}
	if v, ok := m["exclusive"].(bool); ok {
		js.Exclusive = v
	}
	if v, ok := m["requeue"].(bool); ok {
		js.Requeue = v
	}
	if v, ok := m["constraints"].(string); ok {
		js.Constraints = v
	}
	if v, ok := m["ntasks"]; ok {
		js.NTasks = toInt(v)
	}
	if v, ok := m["ntasksPerNode"]; ok {
		js.NTasksPerNode = toInt(v)
	}
	if v, ok := m["gres"].(string); ok {
		js.Gres = v
	}
	if v, ok := m["hold"].(bool); ok {
		js.Hold = v
	}
	if v, ok := m["reservation"].(string); ok {
		js.Reservation = v
	}
	if v, ok := m["licenses"].(string); ok {
		js.Licenses = v
	}
	if v, ok := m["wckey"].(string); ok {
		js.Wckey = v
	}
	if v, ok := m["excludeNodes"].(string); ok {
		js.ExcludeNodes = v
	}
	if v, ok := m["priority"]; ok {
		js.Priority = toInt(v)
	}
	if v, ok := m["nice"]; ok {
		js.Nice = toInt(v)
	}
	if v, ok := m["memoryPerCPU"].(string); ok {
		js.MemoryPerCPU = v
	}
	if v, ok := m["beginTime"].(string); ok {
		js.BeginTime = v
	}
	if v, ok := m["comment"].(string); ok {
		js.Comment = v
	}
	if v, ok := m["distribution"].(string); ok {
		js.Distribution = v
	}
	if v, ok := m["prefer"].(string); ok {
		js.Prefer = v
	}
	if v, ok := m["requiredNodes"].(string); ok {
		js.RequiredNodes = v
	}
	if v, ok := m["standardInput"].(string); ok {
		js.StandardInput = v
	}
	if v, ok := m["container"].(string); ok {
		js.Container = v
	}
	if v, ok := m["threadsPerCore"]; ok {
		js.ThreadsPerCore = toInt(v)
	}
	if v, ok := m["tasksPerCore"]; ok {
		js.TasksPerCore = toInt(v)
	}
	if v, ok := m["tasksPerSocket"]; ok {
		js.TasksPerSocket = toInt(v)
	}
	if v, ok := m["socketsPerNode"]; ok {
		js.SocketsPerNode = toInt(v)
	}
	if v, ok := m["maximumNodes"]; ok {
		js.MaximumNodes = toInt(v)
	}
	if v, ok := m["maximumCPUs"]; ok {
		js.MaximumCPUs = toInt(v)
	}
	if v, ok := m["minimumCPUsPerNode"]; ok {
		js.MinimumCPUsPerNode = toInt(v)
	}
	if v, ok := m["timeMinimum"].(string); ok {
		js.TimeMinimum = v
	}
	if v, ok := m["contiguous"].(bool); ok {
		js.Contiguous = v
	}
	if v, ok := m["overcommit"].(bool); ok {
		js.Overcommit = v
	}
	if v, ok := m["killOnNodeFail"].(bool); ok {
		js.KillOnNodeFail = v
	}
	if v, ok := m["waitAllNodes"].(bool); ok {
		js.WaitAllNodes = v
	}
	if v, ok := m["openMode"].(string); ok {
		js.OpenMode = v
	}
	if v, ok := m["tresPerTask"].(string); ok {
		js.TRESPerTask = v
	}
	if v, ok := m["tresPerSocket"].(string); ok {
		js.TRESPerSocket = v
	}
	if v, ok := m["signal"].(string); ok {
		js.Signal = v
	}
	if v, ok := m["tmpDiskPerNode"]; ok {
		js.TmpDiskPerNode = toInt(v)
	}
	if v, ok := m["deadline"].(string); ok {
		js.Deadline = v
	}
	if v, ok := m["ntasksPerTRES"]; ok {
		js.NTasksPerTRES = toInt(v)
	}
	if v, ok := m["cpuBinding"].(string); ok {
		js.CPUBinding = v
	}
	if v, ok := m["cpuFrequency"].(string); ok {
		js.CPUFrequency = v
	}
	if v, ok := m["network"].(string); ok {
		js.Network = v
	}
	if v, ok := m["x11"].(string); ok {
		js.X11 = v
	}
	if v, ok := m["immediate"].(bool); ok {
		js.Immediate = v
	}
	if v, ok := m["burstBuffer"].(string); ok {
		js.BurstBuffer = v
	}
	if v, ok := m["batchFeatures"].(string); ok {
		js.BatchFeatures = v
	}
	if v, ok := m["tresBind"].(string); ok {
		js.TRESBind = v
	}
	if v, ok := m["tresFreq"].(string); ok {
		js.TRESFreq = v
	}
	if v, ok := m["coreSpecification"]; ok {
		js.CoreSpecification = toInt(v)
	}
	if v, ok := m["threadSpecification"]; ok {
		js.ThreadSpecification = toInt(v)
	}
	if v, ok := m["memoryBinding"].(string); ok {
		js.MemoryBinding = v
	}
	if v, ok := m["minimumCPUs"]; ok {
		js.MinimumCPUs = toInt(v)
	}
	if v, ok := m["tresPerJob"].(string); ok {
		js.TRESPerJob = v
	}
	if v, ok := m["cpusPerTRES"].(string); ok {
		js.CPUsPerTRES = v
	}
	if v, ok := m["memoryPerTRES"].(string); ok {
		js.MemoryPerTRES = v
	}
	if v, ok := m["argv"].(string); ok {
		js.Argv = v
	}
	if v, ok := m["flags"].(string); ok {
		js.Flags = v
	}
	if v, ok := m["profile"].(string); ok {
		js.ProfileTypes = v
	}
	if v, ok := m["cpuBindingFlags"].(string); ok {
		js.CPUBindingFlags = v
	}
	if v, ok := m["memoryBindingType"].(string); ok {
		js.MemoryBindingType = v
	}
	if v, ok := m["requiredSwitches"]; ok {
		js.RequiredSwitches = toInt(v)
	}
	if v, ok := m["waitForSwitch"]; ok {
		js.WaitForSwitch = toInt(v)
	}
	if v, ok := m["clusterConstraint"].(string); ok {
		js.ClusterConstraint = v
	}
	if v, ok := m["clusters"].(string); ok {
		js.Clusters = v
	}
	if v, ok := m["dependencies"]; ok {
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					js.Dependencies = append(js.Dependencies, s)
				}
			}
		}
	}

	return js
}

// ResolveTemplateSources returns the effective template sources.
// If TemplateSources is explicitly set, use it directly.
// Otherwise, derive from ShowBuiltinTemplates for backward compatibility.
// Valid sources: "builtin", "config", "saved"
// Default (no config): all three sources.
func ResolveTemplateSources(cfg *JobSubmissionConfig) []string {
	if cfg != nil && len(cfg.TemplateSources) > 0 {
		return cfg.TemplateSources
	}
	// Backward compat: derive from showBuiltinTemplates
	if cfg != nil && cfg.ShowBuiltinTemplates != nil && !*cfg.ShowBuiltinTemplates {
		return []string{"config", "saved"}
	}
	return []string{"builtin", "config", "saved"}
}

// HasTemplateSource checks if a source is in the resolved sources list
func HasTemplateSource(sources []string, source string) bool {
	for _, s := range sources {
		if s == source {
			return true
		}
	}
	return false
}

// toInt converts a value to int, handling both int and float64 types
// that may result from YAML/JSON unmarshaling.
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}
