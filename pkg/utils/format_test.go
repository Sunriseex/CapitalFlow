package utils

import "testing"

func TestRublesToPositiveKopecksRejectsNonPositiveAmount(t *testing.T) {
	tests := []string{"0", "-1.00"}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			_, err := RublesToPositiveKopecks(tt)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRublesToPositiveKopecks(t *testing.T) {
	got, err := RublesToPositiveKopecks("349.90")
	if err != nil {
		t.Fatalf("parse amount: %v", err)
	}
	if got != 34_990 {
		t.Fatalf("amount = %d, want 34990", got)
	}
}
