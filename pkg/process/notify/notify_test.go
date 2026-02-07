package notify

import (
	"os"
	"path"
	"testing"

	"github.com/spf13/viper"
	. "github.com/ztrade/ztrade/pkg/core"
)

func TestSendNotify(t *testing.T) {
	home, _ := os.UserHomeDir()
	viper.AddConfigPath(path.Join(home, ".config"))
	viper.SetConfigName("ztrade")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		t.Fatal(err)
	}
	n, err := NewNotify(viper.GetViper())
	if err != nil {
		t.Fatal(err.Error())
	}
	err = n.SendNotify(&NotifyEvent{
		Title:   "hello",
		Content: "just a test",
	})
	if err != nil {
		t.Fatal(err.Error())
	}
}
