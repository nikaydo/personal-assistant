package ai

import "testing"

func TestHeuristicRoute_Agent(t *testing.T) {
	route, ok := heuristicRoute("Привет, создай файл hello.txt")
	if !ok {
		t.Fatalf("expected heuristic hit")
	}
	if route != routeAgent {
		t.Fatalf("expected routeAgent, got %s", route)
	}
}

func TestHeuristicRoute_Default(t *testing.T) {
	route, ok := heuristicRoute("Расскажи про алгоритм Dijkstra")
	if ok {
		t.Fatalf("expected no forced heuristic for neutral question, got %s", route)
	}
}

func TestParsePlannerDecision_FromJSONBlock(t *testing.T) {
	d, ok := parsePlannerDecision("prefix {\"route\":\"agent\",\"reason\":\"needs actions\"} suffix")
	if !ok {
		t.Fatalf("expected valid planner decision")
	}
	if d.Route != "agent" {
		t.Fatalf("unexpected route: %s", d.Route)
	}
	if d.Reason != "needs actions" {
		t.Fatalf("unexpected reason: %s", d.Reason)
	}
}
