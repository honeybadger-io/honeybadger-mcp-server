package hbmcp

import (
	"context"
	"reflect"
	"testing"
)

// The spec lets null clear an Ingestion Key's label; the tool passes it on.
func TestUpdateIngestionKeyClearsLabelWithNull(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"k1"}}`)
	result, err := handleUpdateIngestionKey(context.Background(), client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "ingestion_key_id": "k1", "label": nil,
	}))
	mustSucceed(t, result, err)
	if want := map[string]any{"label": nil}; !reflect.DeepEqual(*body, want) {
		t.Errorf("body = %v, want %v", *body, want)
	}
}

func TestUpdateIngestionKeyNeedsSomethingToChange(t *testing.T) {
	result, err := handleUpdateIngestionKey(context.Background(), noRequestClient(t), mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "ingestion_key_id": "k1",
	}))
	if err != nil || !result.IsError {
		t.Errorf("result = %s, want refused", getResultText(result))
	}
}
