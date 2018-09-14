package cli

import (
	"reflect"
	"testing"
)

func TestLoadAccountJSON(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     *ServiceAccount
		wantErr  bool
	}{
		{name: "OK", filename: "account-dummy.json", want: &ServiceAccount{ProjectID: "neco-test", ClientEmail: "neco-test@neco-test.iam.gserviceaccount.com"}, wantErr: false},
		{name: "not found", filename: "not-found.json", want: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadAccountJSON(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadAccountJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadAccountJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
