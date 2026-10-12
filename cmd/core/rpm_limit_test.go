package main

import "testing"

func TestRPMLimitForModel(t *testing.T) {
	pro := map[uint8]int{0: 4000, 1: 2700, 2: 3300, 3: 4000}
	plain := map[uint8]int{0: 3000, 1: 2700, 2: 3000, 3: 3000}
	for vbus, want := range pro {
		if got := rpmLimitForModel("BS2PRO", vbus); got != want {
			t.Fatalf("BS2PRO vbus %d = %d, want %d", vbus, got, want)
		}
	}
	for vbus, want := range plain {
		if got := rpmLimitForModel("BS2", vbus); got != want {
			t.Fatalf("BS2 vbus %d = %d, want %d", vbus, got, want)
		}
	}
}
