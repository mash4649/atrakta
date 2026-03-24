package resolvers_test

import (
	"bytes"
	"testing"

	"github.com/mash4649/atrakta/v0/internal/pipeline"
)

func TestDeterministicReplayOrderedPipeline(t *testing.T) {
	input := pipeline.DefaultInput("preview")

	out1, err := pipeline.ExecuteOrdered("preview", input)
	if err != nil {
		t.Fatalf("first execute error: %v", err)
	}
	out2, err := pipeline.ExecuteOrdered("preview", input)
	if err != nil {
		t.Fatalf("second execute error: %v", err)
	}

	b1, err := pipeline.MarshalStable(out1)
	if err != nil {
		t.Fatalf("marshal1 error: %v", err)
	}
	b2, err := pipeline.MarshalStable(out2)
	if err != nil {
		t.Fatalf("marshal2 error: %v", err)
	}

	if !bytes.Equal(b1, b2) {
		t.Fatalf("pipeline output is not deterministic")
	}
}
