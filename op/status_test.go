package op

import "testing"

func Test_containCommandOption(t *testing.T) {
	type args struct {
		slice      []string
		optionName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"valid",
			args{[]string{"scheduler", "--config", "aaa"}, "--config"},
			true,
		},
		{
			"with =",
			args{[]string{"scheduler", "--config=aaa"}, "--config"},
			true,
		},
		{
			"with space character",
			args{[]string{"scheduler", "--config aaa"}, "--config"},
			true,
		},
		{
			"no content",
			args{[]string{"scheduler", "--option1", "aaa"}, "--config"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containCommandOption(tt.args.slice, tt.args.optionName); got != tt.want {
				t.Errorf("containCommandOption() = %v, want %v", got, tt.want)
			}
		})
	}
}
