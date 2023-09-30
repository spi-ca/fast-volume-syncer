package sys

import (
	"io/fs"
	"testing"
)

func TestUnFilemodeStr(t *testing.T) {
	type args struct {
		modeStr string
	}
	tests := []struct {
		name string
		args args
		want fs.FileMode
	}{
		{
			name: "(empty)",
			args: args{
				modeStr: "",
			},
			want: 0o000,
		},
		{
			name: "-rw-r--r--",
			args: args{
				modeStr: "-rw-r--r--",
			},
			want: 0o644,
		},
		{
			name: "directory 0755",
			args: args{
				modeStr: "drwxr-xr-x",
			},
			want: 0o755 | fs.ModeDir,
		},
		{
			name: "symbolic link",
			args: args{
				modeStr: "lrwxrwxrwx",
			},
			want: 0o777 | fs.ModeSymlink,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnFilemodeStr(tt.args.modeStr); got != tt.want {
				t.Errorf("UnFilemodeStr() = %v, want %v", got, tt.want)
			}
		})
	}
}
