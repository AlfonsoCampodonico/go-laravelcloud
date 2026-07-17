package models

import "testing"

func TestUserInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		in      UserInput
		wantErr bool
	}{
		{"valid", UserInput{Name: "Ada", Email: "ada@example.com"}, false},
		{"trims and accepts", UserInput{Name: "  Ada  ", Email: "ada@example.com"}, false},
		{"empty name", UserInput{Name: "   ", Email: "ada@example.com"}, true},
		{"missing email", UserInput{Name: "Ada", Email: ""}, true},
		{"bad email", UserInput{Name: "Ada", Email: "not-an-email"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserInput_ValidateRejectsLongName(t *testing.T) {
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'a'
	}
	in := UserInput{Name: string(long), Email: "ada@example.com"}
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for 101-char name, got nil")
	}
}
