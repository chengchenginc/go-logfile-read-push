package main

import (
	"fmt"
	components "github.com/chengchenginc/go-logfile-read-push/components"
	"github.com/chengchenginc/go-logfile-read-push/config"
	"github.com/robfig/cron"
	"os"
	_ "time"
)

func main() {
	fmt.Println("=====run start!=====")
	config.LoadConfig("config/config.toml")
	RPer, err := components.NewReadPusher(config.Config.Redis, config.Config.LogFile)
	if err != nil {
		fmt.Println("can not load readpusher")
		os.Exit(-1)
	}
	c := cron.New()
	c.AddFunc("*/5 * * * * *", func() {
		RPer.ReadLines()
	})
	c.Start()
	//c.Stop()
	//time.Sleep(time.Minute)
	select {}
}
