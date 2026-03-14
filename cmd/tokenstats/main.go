package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	Cost             float64 `json:"cost"`
}

type turn struct {
	Answer struct {
		Usage usage `json:"Usage"`
	} `json:"answer"`
}

type state struct {
	ShortTerm []turn `json:"short_term"`
}

func main() {
	path := flag.String("state", "./data/memory_state.json", "path to memory_state.json")
	flag.Parse()
	raw, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read state file: %v\n", err)
		os.Exit(1)
	}
	var s state
	if err := json.Unmarshal(raw, &s); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal state file: %v\n", err)
		os.Exit(1)
	}
	if len(s.ShortTerm) == 0 {
		fmt.Println("turns=0")
		return
	}

	var (
		promptTotal     int
		completionTotal int
		totalTotal      int
		costTotal       float64
	)
	for _, t := range s.ShortTerm {
		u := t.Answer.Usage
		promptTotal += u.PromptTokens
		completionTotal += u.CompletionTokens
		totalTotal += u.TotalTokens
		costTotal += u.Cost
	}
	turns := len(s.ShortTerm)
	fmt.Printf("turns=%d\n", turns)
	fmt.Printf("avg_prompt_tokens=%.1f\n", float64(promptTotal)/float64(turns))
	fmt.Printf("avg_completion_tokens=%.1f\n", float64(completionTotal)/float64(turns))
	fmt.Printf("avg_total_tokens=%.1f\n", float64(totalTotal)/float64(turns))
	if totalTotal > 0 {
		fmt.Printf("prompt_share=%.1f%%\n", float64(promptTotal)*100/float64(totalTotal))
		fmt.Printf("completion_share=%.1f%%\n", float64(completionTotal)*100/float64(totalTotal))
	}
	if completionTotal > 0 {
		fmt.Printf("waste_ratio_prompt_to_completion=%.2f\n", float64(promptTotal)/float64(completionTotal))
	}
	fmt.Printf("avg_cost=%.8f\n", costTotal/float64(turns))
	fmt.Printf("total_cost=%.8f\n", costTotal)
}
