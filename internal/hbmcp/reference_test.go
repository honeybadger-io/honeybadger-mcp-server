package hbmcp

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func referenceRequest(topics ...string) mcp.CallToolRequest {
	args := map[string]interface{}{}
	if topics != nil {
		list := make([]interface{}, len(topics))
		for i, t := range topics {
			list[i] = t
		}
		args["topics"] = list
	}
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: args},
	}
}

func TestHandleGetReference_Index(t *testing.T) {
	result, err := handleGetReference(context.Background(), referenceRequest())
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	for _, topic := range referenceTopics {
		if !strings.Contains(resultText, topic.name) {
			t.Errorf("index should list topic %q", topic.name)
		}
		if strings.Contains(resultText, topic.content) {
			t.Errorf("index should not include full content of topic %q", topic.name)
		}
	}
}

func TestHandleGetReference_SingleTopic(t *testing.T) {
	result, err := handleGetReference(context.Background(), referenceRequest("badgerql"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	expectedMarkers := []string{"BadgerQL", "fields", "filter", "stats", "query_insights"}
	for _, marker := range expectedMarkers {
		if !strings.Contains(resultText, marker) {
			t.Errorf("badgerql topic should contain %q", marker)
		}
	}
	if strings.Contains(resultText, "trigger_config") {
		t.Error("badgerql topic should not contain alarm content")
	}
}

func TestHandleGetReference_MultipleTopicsStableOrder(t *testing.T) {
	// Request in reverse order; composition must follow canonical order.
	result, err := handleGetReference(context.Background(), referenceRequest("charts", "badgerql"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	badgerqlPos := strings.Index(resultText, "# BadgerQL Query Reference")
	chartsPos := strings.Index(resultText, "# Charts & Visualizations")
	if badgerqlPos == -1 || chartsPos == -1 {
		t.Fatal("expected both topic headings in result")
	}
	if badgerqlPos > chartsPos {
		t.Error("badgerql should precede charts regardless of request order")
	}
}

func TestHandleGetReference_DuplicateTopics(t *testing.T) {
	result, err := handleGetReference(context.Background(), referenceRequest("alarms", "alarms"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	if strings.Count(resultText, "# Alarms") != 1 {
		t.Error("duplicate topic names should be deduplicated")
	}
}

func TestHandleGetReference_All(t *testing.T) {
	result, err := handleGetReference(context.Background(), referenceRequest("all"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	for _, topic := range referenceTopics {
		if !strings.Contains(resultText, topic.content) {
			t.Errorf("\"all\" should include full content of topic %q", topic.name)
		}
	}
}

func TestHandleGetReference_UnknownTopic(t *testing.T) {
	result, err := handleGetReference(context.Background(), referenceRequest("badgerql", "nonsense"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for unknown topic")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "nonsense") {
		t.Error("error should name the unknown topic")
	}
	if !strings.Contains(resultText, "badgerql") {
		t.Error("error should list valid topic names")
	}
}

func TestReferenceTopics_NonEmpty(t *testing.T) {
	for _, topic := range referenceTopics {
		if strings.TrimSpace(topic.content) == "" {
			t.Errorf("topic %q has empty content", topic.name)
		}
		if !strings.HasPrefix(topic.content, "# ") {
			t.Errorf("topic %q should start with an H1 heading", topic.name)
		}
	}
}

func TestServerInstructions(t *testing.T) {
	instructions := ServerInstructions()
	for _, topic := range referenceTopics {
		if !strings.Contains(instructions, topic.name) {
			t.Errorf("instructions should mention topic %q", topic.name)
		}
	}
	if !strings.Contains(instructions, "get_reference") {
		t.Error("instructions should mention the get_reference tool")
	}
	// Instructions are always-on context for every session — keep them small.
	if len(instructions) > 2500 {
		t.Errorf("instructions too long: %d bytes", len(instructions))
	}
}
