package main

import "testing"

func TestAcceptedWindowSize(t *testing.T) {
	if _, _, ok := acceptedWindowSize(windowDefaultWidth-1, windowDefaultHeight); ok {
		t.Fatal("width below minimum was accepted")
	}
	if _, _, ok := acceptedWindowSize(windowDefaultWidth, windowDefaultHeight-1); ok {
		t.Fatal("height below minimum was accepted")
	}
	w, h, ok := acceptedWindowSize(1200, 800)
	if !ok || w != 1200 || h != 800 {
		t.Fatalf("got %d %d %v", w, h, ok)
	}
}
