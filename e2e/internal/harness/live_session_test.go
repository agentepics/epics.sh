package harness

import "testing"

func TestNormalizeLiveSessionText(t *testing.T) {
	input := " TURN1_OK \r\n SESSION-NONCE-7319\tDONE "
	got := normalizeLiveSessionText(input)
	want := "turn1_oksession-nonce-7319done"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestWithDefaultLiveSessionSpecRejectsEchoMarker(t *testing.T) {
	_, err := withDefaultLiveSessionSpec(&LiveSessionSpec{
		Turns: []LiveSessionTurn{
			{
				Name:     "turn1",
				Prompt:   "Reply exactly TURN1_OK SESSION-NONCE-7319.",
				Expected: "TURN1_OK SESSION-NONCE-7319",
			},
		},
	})
	if err == nil {
		t.Fatal("expected prompt/expected validation error")
	}
}

func TestWithDefaultLiveSessionSpecAppliesDefaults(t *testing.T) {
	spec, err := withDefaultLiveSessionSpec(&LiveSessionSpec{
		Turns: []LiveSessionTurn{
			{
				Name:     "turn1",
				Prompt:   "Reply with the uppercase success token for the first turn.",
				Expected: "TURN1_OK",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.ArtifactDir != ".claude-live-session" {
		t.Fatalf("expected default artifact dir, got %q", spec.ArtifactDir)
	}
	if spec.BootstrapPrompt != "Respond exactly PREPARED" {
		t.Fatalf("unexpected bootstrap prompt: %q", spec.BootstrapPrompt)
	}
	if spec.BootstrapExpect != "PREPARED" {
		t.Fatalf("unexpected bootstrap expect: %q", spec.BootstrapExpect)
	}
}
