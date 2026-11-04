package discovery

import "testing"

func TestPlanFiveInterests(t *testing.T) {
	planner := QueryPlanner{}
	queries, err := planner.Plan([]string{"ai-agents", "local-llms", "distributed-systems", "developer-tools", "machine-learning"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(queries) != 5 {
		t.Fatalf("queries = %d, want 5", len(queries))
	}
	for i, want := range []string{"ai agents", "local llms", "distributed systems", "developer tools", "machine learning"} {
		if queries[i].Text != want {
			t.Fatalf("query %d text = %q, want %q", i, queries[i].Text, want)
		}
		if queries[i].Source != SourceSearch {
			t.Fatalf("query %d source = %q", i, queries[i].Source)
		}
		if queries[i].MaxResults != 50 || queries[i].Days != 30 {
			t.Fatalf("query %d = %+v", i, queries[i])
		}
	}
}

func TestPlanDeterministic(t *testing.T) {
	planner := QueryPlanner{MaxResults: 25, Days: 7}
	first, err := planner.Plan([]string{"go", "rust"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	second, err := planner.Plan([]string{"go", "rust"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("lengths = %d, %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("query %d differs: %+v vs %+v", i, first[i], second[i])
		}
	}
	if first[0].MaxResults != 25 || first[0].Days != 7 {
		t.Fatalf("overrides = %+v", first[0])
	}
}

func TestPlanUnhappy(t *testing.T) {
	tests := []struct {
		name      string
		interests []string
	}{
		{name: "empty", interests: nil},
		{name: "too many", interests: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}},
		{name: "blank", interests: []string{"go", " "}},
		{name: "duplicate", interests: []string{"go", " GO "}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := (QueryPlanner{}).Plan(tt.interests); err == nil {
				t.Fatal("Plan = nil, want an error")
			}
		})
	}
}

func TestQueryText(t *testing.T) {
	query := Query{Interest: "ai-agents"}
	if got := query.SearchText(); got != "ai agents" {
		t.Fatalf("SearchText = %q", got)
	}
	query = Query{Interest: "  Local_LLMs  "}
	if got := query.SearchText(); got != "local llms" {
		t.Fatalf("SearchText = %q", got)
	}
	if got := (Query{}).SearchText(); got != "" {
		t.Fatalf("empty SearchText = %q", got)
	}
}
