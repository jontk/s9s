package views

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ui/styles"
	"github.com/rivo/tview"
)

// SearchResult represents a single search result
type SearchResult struct {
	Type        string // "job", "node", "partition", etc.
	ID          string
	Name        string
	Description string
	Score       int // Relevance score
	Data        interface{}
}

// GlobalSearch provides global search functionality across all data types
type GlobalSearch struct {
	client       dao.SlurmClient
	app          *tview.Application
	pages        *tview.Pages
	searchInput  *tview.InputField
	resultsList  *tview.List
	results      []SearchResult
	mu           sync.RWMutex
	searchCancel chan struct{}
	onSelect     func(result SearchResult)
}

// NewGlobalSearch creates a new global search component
func NewGlobalSearch(client dao.SlurmClient, app *tview.Application) *GlobalSearch {
	gs := &GlobalSearch{
		client:  client,
		app:     app,
		results: []SearchResult{},
	}

	// Create search input with styled colors for visibility across themes
	gs.searchInput = styles.NewStyledInputField().
		SetLabel("Search: ").
		SetFieldWidth(50).
		SetChangedFunc(gs.onSearchChange).
		SetPlaceholder("Type to search jobs, nodes, users, etc...")

	// Create results list
	gs.resultsList = tview.NewList()

	return gs
}

// Show displays the global search interface
func (gs *GlobalSearch) Show(pages *tview.Pages, onSelect func(result SearchResult)) {
	gs.pages = pages
	gs.onSelect = onSelect

	// Create search container
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(gs.searchInput, 3, 0, true).
		AddItem(gs.resultsList, 0, 1, false)

	container.SetBorder(true).
		SetTitle(" Global Search (Ctrl+F) ").
		SetTitleAlign(tview.AlignCenter)

	// Help text
	helpText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]Enter[white] Select | [yellow]↑↓[white] Navigate | [yellow]ESC[white] Close | [yellow]Tab[white] Focus results").
		SetTextAlign(tview.AlignCenter)

	// Full layout
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(container, 0, 1, true).
		AddItem(helpText, 1, 0, false)

	// Create centered modal
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(layout, 30, 1, true).
			AddItem(nil, 0, 1, false), 80, 1, true).
		AddItem(nil, 0, 1, false)

	// Handle navigation
	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			gs.cancelSearch()
			pages.RemovePage("global-search")
			return nil
		case tcell.KeyTab:
			// Toggle focus between input and results
			if gs.searchInput.HasFocus() {
				gs.app.SetFocus(gs.resultsList)
			} else {
				gs.app.SetFocus(gs.searchInput)
			}
			return nil
		case tcell.KeyEnter:
			// Handle Enter from results list
			if gs.resultsList.HasFocus() {
				idx := gs.resultsList.GetCurrentItem()
				if idx >= 0 && idx < len(gs.results) {
					result := gs.results[idx]
					// Remove the modal first, then call the callback
					// This is safe because we're in an event handler - direct primitive
					// manipulation is allowed, but QueueUpdateDraw would deadlock
					pages.RemovePage("global-search")
					if gs.onSelect != nil {
						gs.onSelect(result)
					}
					return nil
				}
			}
			return nil
		}
		return event
	})

	// Handle Enter key in search input
	gs.searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			// Select first result if available
			if len(gs.results) > 0 && gs.onSelect != nil {
				result := gs.results[0]
				// Remove the modal first, then call the callback
				// Direct manipulation is safe in event handlers - no queueing needed
				pages.RemovePage("global-search")
				gs.onSelect(result)
			}
			return nil
		}
		return event
	})

	pages.AddPage("global-search", modal, true, true)
	gs.app.SetFocus(gs.searchInput)
}

// onSearchChange handles search input changes
func (gs *GlobalSearch) onSearchChange(text string) {
	// Cancel previous search
	gs.cancelSearch()

	if len(text) < 2 {
		gs.clearResults()
		return
	}

	// Start new search
	gs.searchCancel = make(chan struct{})
	go gs.performSearch(text, gs.searchCancel)
}

// performSearch performs the actual search across all data types
func (gs *GlobalSearch) performSearch(query string, cancel chan struct{}) {
	var wg sync.WaitGroup
	results := make(chan SearchResult, 100)

	// Search jobs
	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.searchJobs(query, results, cancel)
	}()

	// Search nodes
	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.searchNodes(query, results, cancel)
	}()

	// Search partitions
	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.searchPartitions(query, results, cancel)
	}()

	// Search users
	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.searchUsers(query, results, cancel)
	}()

	// Search reservations
	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.searchReservations(query, results, cancel)
	}()

	// Search accounts
	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.searchAccounts(query, results, cancel)
	}()

	// Search QoS
	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.searchQoS(query, results, cancel)
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Update UI with results
	var allResults []SearchResult
	for result := range results {
		select {
		case <-cancel:
			return
		default:
			allResults = append(allResults, result)
		}
	}

	// Sort by relevance score
	gs.sortResultsByScore(allResults)

	// Update UI
	gs.app.QueueUpdateDraw(func() {
		gs.updateResultsList(allResults)
	})
}

// searchJobs searches through jobs
func (gs *GlobalSearch) searchJobs(query string, results chan<- SearchResult, cancel chan struct{}) {
	jobs, err := gs.client.Jobs().List(&dao.ListJobsOptions{Limit: 100})
	if err != nil {
		return
	}

	queryLower := strings.ToLower(query)
	jobCount := 0
	for _, job := range jobs.Jobs {
		select {
		case <-cancel:
			return
		default:
			score := gs.calculateJobScore(job, queryLower)
			if score > 0 {
				jobCount++
				// Format: Job ID as name, with user/state/partition in description
				displayName := fmt.Sprintf("%s (%s)", job.ID, job.Name)
				results <- SearchResult{
					Type:        "job",
					ID:          job.ID,
					Name:        displayName,
					Description: fmt.Sprintf("User: %s | State: %s | Partition: %s", job.User, job.State, job.Partition),
					Score:       score,
					Data:        job,
				}
			}
		}
	}
}

// calculateJobScore calculates search relevance score for a job
func (gs *GlobalSearch) calculateJobScore(job *dao.Job, queryLower string) int {
	score := 0

	if strings.Contains(strings.ToLower(job.ID), queryLower) {
		score += 10
	}
	if strings.Contains(strings.ToLower(job.Name), queryLower) {
		score += 8
	}
	if strings.Contains(strings.ToLower(job.User), queryLower) {
		score += 5
	}
	if strings.Contains(strings.ToLower(job.Partition), queryLower) {
		score += 3
	}
	if strings.Contains(strings.ToLower(job.State), queryLower) {
		score += 3
	}

	return score
}

// searchNodes searches through nodes
func (gs *GlobalSearch) searchNodes(query string, results chan<- SearchResult, cancel chan struct{}) {
	nodes, err := gs.client.Nodes().List(&dao.ListNodesOptions{})
	if err != nil {
		return
	}

	queryLower := strings.ToLower(query)
	nodeCount := 0
	for _, node := range nodes.Nodes {
		select {
		case <-cancel:
			return
		default:
			score := gs.calculateNodeScore(node, queryLower)
			if score > 0 {
				nodeCount++
				partitions := strings.Join(node.Partitions, ",")
				results <- SearchResult{
					Type:        "node",
					ID:          node.Name,
					Name:        node.Name,
					Description: fmt.Sprintf("State: %s | Partitions: %s | CPUs: %d", node.State, partitions, node.CPUsTotal),
					Score:       score,
					Data:        node,
				}
			}
		}
	}
}

// calculateNodeScore calculates search relevance score for a node
func (gs *GlobalSearch) calculateNodeScore(node *dao.Node, queryLower string) int {
	score := 0

	if strings.Contains(strings.ToLower(node.Name), queryLower) {
		score += 10
	}
	if strings.Contains(strings.ToLower(node.State), queryLower) {
		score += 5
	}

	for _, feature := range node.Features {
		if strings.Contains(strings.ToLower(feature), queryLower) {
			score += 3
			break
		}
	}

	for _, partition := range node.Partitions {
		if strings.Contains(strings.ToLower(partition), queryLower) {
			score += 3
			break
		}
	}

	return score
}

// searchPartitions searches through partitions
func (gs *GlobalSearch) searchPartitions(query string, results chan<- SearchResult, cancel chan struct{}) {
	partitions, err := gs.client.Partitions().List()
	if err != nil {
		return
	}

	queryLower := strings.ToLower(query)
	for _, partition := range partitions.Partitions {
		select {
		case <-cancel:
			return
		default:
			score := 0

			// Check partition name
			if strings.Contains(strings.ToLower(partition.Name), queryLower) {
				score += 10
			}
			// Check state
			if strings.Contains(strings.ToLower(partition.State), queryLower) {
				score += 5
			}
			// Check QoS
			for _, qos := range partition.QOS {
				if strings.Contains(strings.ToLower(qos), queryLower) {
					score += 3
					break
				}
			}

			if score > 0 {
				results <- SearchResult{
					Type:        "partition",
					ID:          partition.Name,
					Name:        partition.Name,
					Description: fmt.Sprintf("State: %s | Nodes: %d | CPUs: %d", partition.State, partition.TotalNodes, partition.TotalCPUs),
					Score:       score,
					Data:        partition,
				}
			}
		}
	}
}

// searchUsers searches through users
func (gs *GlobalSearch) searchUsers(query string, results chan<- SearchResult, cancel chan struct{}) {
	users, err := gs.client.Users().List()
	if err != nil {
		return
	}

	queryLower := strings.ToLower(query)
	for _, user := range users.Users {
		select {
		case <-cancel:
			return
		default:
			score := 0

			// Check user name
			if strings.Contains(strings.ToLower(user.Name), queryLower) {
				score += 10
			}
			// Check default account
			if strings.Contains(strings.ToLower(user.DefaultAccount), queryLower) {
				score += 5
			}
			// Check accounts
			for _, account := range user.Accounts {
				if strings.Contains(strings.ToLower(account), queryLower) {
					score += 3
					break
				}
			}

			if score > 0 {
				accounts := strings.Join(user.Accounts, ",")
				results <- SearchResult{
					Type:        "user",
					ID:          user.Name,
					Name:        user.Name,
					Description: fmt.Sprintf("Default Account: %s | Accounts: %s", user.DefaultAccount, accounts),
					Score:       score,
					Data:        user,
				}
			}
		}
	}
}

// searchReservations searches through reservations
func (gs *GlobalSearch) searchReservations(query string, results chan<- SearchResult, cancel chan struct{}) {
	reservations, err := gs.client.Reservations().List()
	if err != nil {
		return
	}

	queryLower := strings.ToLower(query)
	for _, reservation := range reservations.Reservations {
		select {
		case <-cancel:
			return
		default:
			score := 0

			// Check reservation name
			if strings.Contains(strings.ToLower(reservation.Name), queryLower) {
				score += 10
			}
			// Check state
			if strings.Contains(strings.ToLower(reservation.State), queryLower) {
				score += 5
			}
			// Check users
			for _, user := range reservation.Users {
				if strings.Contains(strings.ToLower(user), queryLower) {
					score += 3
					break
				}
			}
			// Check accounts
			for _, account := range reservation.Accounts {
				if strings.Contains(strings.ToLower(account), queryLower) {
					score += 3
					break
				}
			}

			if score > 0 {
				results <- SearchResult{
					Type:        "reservation",
					ID:          reservation.Name,
					Name:        reservation.Name,
					Description: fmt.Sprintf("State: %s | Nodes: %d | Users: %s", reservation.State, reservation.NodeCount, strings.Join(reservation.Users, ",")),
					Score:       score,
					Data:        reservation,
				}
			}
		}
	}
}

// searchAccounts searches through accounts
func (gs *GlobalSearch) searchAccounts(query string, results chan<- SearchResult, cancel chan struct{}) {
	accounts, err := gs.client.Accounts().List()
	if err != nil {
		return
	}

	queryLower := strings.ToLower(query)
	for _, account := range accounts.Accounts {
		select {
		case <-cancel:
			return
		default:
			score := 0

			// Check account name
			if strings.Contains(strings.ToLower(account.Name), queryLower) {
				score += 10
			}
			// Check description
			if strings.Contains(strings.ToLower(account.Description), queryLower) {
				score += 5
			}
			// Check organization
			if strings.Contains(strings.ToLower(account.Organization), queryLower) {
				score += 3
			}
			// Check parent
			if strings.Contains(strings.ToLower(account.Parent), queryLower) {
				score += 3
			}

			if score > 0 {
				results <- SearchResult{
					Type:        "account",
					ID:          account.Name,
					Name:        account.Name,
					Description: fmt.Sprintf("Org: %s | Parent: %s | Default QoS: %s", account.Organization, account.Parent, account.DefaultQoS),
					Score:       score,
					Data:        account,
				}
			}
		}
	}
}

// searchQoS searches through QoS entries
func (gs *GlobalSearch) searchQoS(query string, results chan<- SearchResult, cancel chan struct{}) {
	qosList, err := gs.client.QoS().List()
	if err != nil {
		return
	}

	queryLower := strings.ToLower(query)
	for _, qos := range qosList.QoS {
		select {
		case <-cancel:
			return
		default:
			score := 0

			// Check QoS name
			if strings.Contains(strings.ToLower(qos.Name), queryLower) {
				score += 10
			}
			// Check preempt mode
			if strings.Contains(strings.ToLower(qos.PreemptMode), queryLower) {
				score += 5
			}
			// Check flags
			for _, flag := range qos.Flags {
				if strings.Contains(strings.ToLower(flag), queryLower) {
					score += 3
					break
				}
			}

			if score > 0 {
				results <- SearchResult{
					Type:        "qos",
					ID:          qos.Name,
					Name:        qos.Name,
					Description: fmt.Sprintf("Priority: %d | Preempt: %s | Max Jobs/User: %d", qos.Priority, qos.PreemptMode, qos.MaxJobsPerUser),
					Score:       score,
					Data:        qos,
				}
			}
		}
	}
}

// sortResultsByScore sorts results by relevance score
func (gs *GlobalSearch) sortResultsByScore(results []SearchResult) {
	// Simple bubble sort for now
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// updateResultsList updates the results list UI
func (gs *GlobalSearch) updateResultsList(results []SearchResult) {
	gs.mu.Lock()
	gs.results = results
	gs.mu.Unlock()

	gs.resultsList.Clear()

	if len(results) == 0 {
		gs.resultsList.AddItem("No results found", "", 0, nil)
		return
	}

	// Add results to list (limit to top 20)
	maxResults := 20
	if len(results) < maxResults {
		maxResults = len(results)
	}

	for i := 0; i < maxResults; i++ {
		result := results[i]
		icon := gs.getIconForType(result.Type)
		title := fmt.Sprintf("%s %s - %s", icon, result.Type, result.Name)

		gs.resultsList.AddItem(title, result.Description, 0, func() {
			// Item selected
		})
	}

	if len(results) > maxResults {
		gs.resultsList.AddItem(
			fmt.Sprintf("... and %d more results", len(results)-maxResults),
			"Refine your search for better results",
			0,
			nil,
		)
	}
}

// getIconForType returns an icon for the result type
func (gs *GlobalSearch) getIconForType(resultType string) string {
	switch resultType {
	case "job":
		return "📋"
	case "node":
		return "🖥️"
	case "partition":
		return "📁"
	case "user":
		return "👤"
	case "account":
		return "🏢"
	case "qos":
		return "⭐"
	default:
		return "•"
	}
}

// clearResults clears the results list
func (gs *GlobalSearch) clearResults() {
	gs.mu.Lock()
	gs.results = []SearchResult{}
	gs.mu.Unlock()

	gs.resultsList.Clear()
	gs.resultsList.AddItem("Type to search...", "Minimum 2 characters", 0, nil)
}

// cancelSearch cancels the current search operation
func (gs *GlobalSearch) cancelSearch() {
	if gs.searchCancel != nil {
		close(gs.searchCancel)
		gs.searchCancel = nil
	}
}
