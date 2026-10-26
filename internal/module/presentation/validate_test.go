package presentation

import (
	"strings"
	"testing"

	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
)

func validRequest() model.CreatePresentionRequestDTO {
	return model.CreatePresentionRequestDTO{
		Title: "  Quiz  ",
		Polls: []model.Poll{{
			Question: "  Red planet? ",
			Options:  []model.Option{{Key: "A", Value: " Venus "}, {Key: "B", Value: "Mars", Correct: true}},
		}},
	}
}

func TestNormalizeTrimsValidInput(t *testing.T) {
	req := validRequest()
	if err := normalize(&req); err != nil {
		t.Fatal(err)
	}
	if req.Title != "Quiz" || req.Polls[0].Question != "Red planet?" || req.Polls[0].Options[0].Value != "Venus" {
		t.Fatalf("not trimmed: %+v", req)
	}
}

func TestNormalizeRejectsBrokenRules(t *testing.T) {
	cases := map[string]func(*model.CreatePresentionRequestDTO){
		"no polls":       func(r *model.CreatePresentionRequestDTO) { r.Polls = nil },
		"empty question": func(r *model.CreatePresentionRequestDTO) { r.Polls[0].Question = "  " },
		"one option":     func(r *model.CreatePresentionRequestDTO) { r.Polls[0].Options = r.Polls[0].Options[:1] },
		"duplicate key":  func(r *model.CreatePresentionRequestDTO) { r.Polls[0].Options[1].Key = "A" },
		"empty answer":   func(r *model.CreatePresentionRequestDTO) { r.Polls[0].Options[1].Value = "" },
		"long title":     func(r *model.CreatePresentionRequestDTO) { r.Title = strings.Repeat("x", 121) },
		"too many options": func(r *model.CreatePresentionRequestDTO) {
			for _, k := range []string{"C", "D", "E", "F", "G"} {
				r.Polls[0].Options = append(r.Polls[0].Options, model.Option{Key: k, Value: k})
			}
		},
	}
	for name, breakIt := range cases {
		req := validRequest()
		breakIt(&req)
		err := normalize(&req)
		if _, ok := err.(model.ValidationError); !ok {
			t.Errorf("%s: got %v, want a ValidationError", name, err)
		}
	}
}
