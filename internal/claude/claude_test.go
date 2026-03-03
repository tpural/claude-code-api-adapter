package claude

import (
	"testing"
)

func TestExtractTextDelta_ContentBlockDelta(t *testing.T) {
	event := `{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`

	text, ok := ExtractTextDelta([]byte(event))
	if !ok {
		t.Fatal("expected text_delta to be extracted")
	}
	if text != "Hello" {
		t.Errorf("expected 'Hello', got %q", text)
	}
}

func TestExtractTextDelta_StreamEvent(t *testing.T) {
	event := `{"type":"stream_event","event":{"delta":{"type":"text_delta","text":"World"}}}`

	text, ok := ExtractTextDelta([]byte(event))
	if !ok {
		t.Fatal("expected text_delta to be extracted")
	}
	if text != "World" {
		t.Errorf("expected 'World', got %q", text)
	}
}

func TestExtractTextDelta_NonTextEvent(t *testing.T) {
	event := `{"type":"content_block_start","content_block":{"type":"text"}}`

	_, ok := ExtractTextDelta([]byte(event))
	if ok {
		t.Error("expected no text_delta from content_block_start")
	}
}

func TestExtractTextDelta_InvalidJSON(t *testing.T) {
	_, ok := ExtractTextDelta([]byte("not json"))
	if ok {
		t.Error("expected no text_delta from invalid JSON")
	}
}

func TestExtractTextDelta_WrongDeltaType(t *testing.T) {
	event := `{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{}"}}`

	_, ok := ExtractTextDelta([]byte(event))
	if ok {
		t.Error("expected no text_delta from input_json_delta")
	}
}

func TestExtractTextDelta_EmptyText(t *testing.T) {
	event := `{"type":"content_block_delta","delta":{"type":"text_delta","text":""}}`

	text, ok := ExtractTextDelta([]byte(event))
	if !ok {
		t.Fatal("expected ok=true for text_delta with empty text")
	}
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}
