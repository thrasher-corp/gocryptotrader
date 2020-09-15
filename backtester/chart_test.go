package backtest

import "testing"

// func TestGenerateOutput(t *testing.T) {
// 	err := GenerateOutput([]byte{})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

func TestGenerateOutput(t *testing.T) {
	type args struct {
		result Results
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Valid",
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := GenerateOutput(tt.args.result); (err != nil) != tt.wantErr {
				t.Errorf("GenerateOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
