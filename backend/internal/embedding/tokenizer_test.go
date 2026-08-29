package embedding_test

import (
	"testing"
	"transverse/internal/embedding"
)

func TestTokenizer_BasicTokenize(t *testing.T) {
	tok, err := embedding.NewTokenizer("")
	if err != nil {
		t.Fatalf("failed to initialize tokenizer: %v", err)
	}

	text := "Binary Search Tree in C++"
	encoded := tok.Tokenize(text)

	if len(encoded.InputIDs) < 2 {
		t.Fatalf("expected at least CLS and SEP tokens, got %d", len(encoded.InputIDs))
	}

	// First token must be CLS, last token must be SEP
	if encoded.InputIDs[0] != embedding.DefaultClsID {
		t.Errorf("expected first token to be CLS (%d), got %d", embedding.DefaultClsID, encoded.InputIDs[0])
	}
	if encoded.InputIDs[len(encoded.InputIDs)-1] != embedding.DefaultSepID {
		t.Errorf("expected last token to be SEP (%d), got %d", embedding.DefaultSepID, encoded.InputIDs[len(encoded.InputIDs)-1])
	}

	// Verify AttentionMask
	if len(encoded.AttentionMask) != len(encoded.InputIDs) {
		t.Errorf("attention mask length mismatch")
	}
	for _, m := range encoded.AttentionMask {
		if m != 1 {
			t.Errorf("expected attention mask values to be 1 for non-padded tokens, got %d", m)
		}
	}
}

func TestTokenizer_BatchTokenize(t *testing.T) {
	tok, err := embedding.NewTokenizer("")
	if err != nil {
		t.Fatalf("failed to initialize tokenizer: %v", err)
	}

	texts := []string{
		"Short text",
		"A much longer problem description involving dynamic programming and segment trees",
	}

	batch, maxLen := tok.TokenizeBatch(texts)
	if len(batch) != 2 {
		t.Fatalf("expected batch size 2, got %d", len(batch))
	}

	if maxLen <= 0 {
		t.Fatalf("expected maxLen > 0, got %d", maxLen)
	}

	// Verify all items padded to maxLen
	for i, item := range batch {
		if len(item.InputIDs) != maxLen {
			t.Errorf("batch item %d InputIDs length %d != maxLen %d", i, len(item.InputIDs), maxLen)
		}
		if len(item.AttentionMask) != maxLen {
			t.Errorf("batch item %d AttentionMask length %d != maxLen %d", i, len(item.AttentionMask), maxLen)
		}
	}
}

func TestTokenizer_Truncation(t *testing.T) {
	tok, err := embedding.NewTokenizer("")
	if err != nil {
		t.Fatalf("failed to initialize tokenizer: %v", err)
	}
	tok.SetMaxLength(10)

	longText := "word "
	for i := 0; i < 50; i++ {
		longText += "tree graph binary search "
	}

	encoded := tok.Tokenize(longText)
	if len(encoded.InputIDs) > 10 {
		t.Errorf("expected max length <= 10, got %d", len(encoded.InputIDs))
	}
	if encoded.InputIDs[0] != embedding.DefaultClsID {
		t.Errorf("expected CLS at start")
	}
	if encoded.InputIDs[len(encoded.InputIDs)-1] != embedding.DefaultSepID {
		t.Errorf("expected SEP at end")
	}
}
