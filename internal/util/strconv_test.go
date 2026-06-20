// Package util verifies small parsing helpers used by scanner-heavy code paths.
package util

import "testing"

// TestSimpleStrconv checks the digit-scraping integer parser on empty, positive, and negative inputs.
func TestSimpleStrconv(t *testing.T) {
	// args holds one input byte slice for the table-driven parser test.
	type args struct {
		// data is the raw byte slice passed to SimpleStrconv.
		data []byte
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "convert from nil",
			args: args{
				data: nil,
			},
			want: -1,
		},
		{
			name: "convert from 13",
			args: args{
				data: []byte("13"),
			},
			want: 13,
		},
		{
			name: "convert from -99",
			args: args{
				data: []byte("-99"),
			},
			want: -99,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SimpleStrconv(tt.args.data); got != tt.want {
				t.Errorf("SimpleStrconv() = %v, want %v", got, tt.want)
			}
		})
	}
}
