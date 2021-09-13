package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/config"

	"github.com/robfig/cron/v3"
	"github.com/ws6/klib"
	"github.com/ws6/msi"
)

func makeQuery() msi.M {
	return msi.M{
		`release_at`: msi.M{
			msi.GTE: time.Now(),
		},
	}
}

func getLimit() int {
	return 100
}

//scan db and then run the delay publish
func RunJob(ctx context.Context, db *msi.Msi, msgChan chan *klib.Message) error {
	messageTable := db.GetTable(`message`)
	if messageTable == nil {
		return fmt.Errorf(`no message table`)
	}
	c := messageTable.Find(makeQuery()).CtxChan(ctx, getLimit())
	for msg := range c {
		//TODO republish
		body, err := msi.ToString(msg[`body`])
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		kmsg := new(klib.Message)
		if err := json.Unmarshal([]byte(body), kmsg); err != nil {
			fmt.Println(err.Error())
			continue
		}

		msgChan <- kmsg
		//deletet msg

		if err := messageTable.Remove(msi.M{`id`: msg[`id`]}); err != nil {
			fmt.Println(err.Error())
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			continue
		}

	}
	return nil
}

func RunSchedules(ctx context.Context, db *msi.Msi, cfgr config.Configer, msgChan chan *klib.Message) error {
	cfg, err := cfgr.GetSection(`cron`)
	if err != nil {
		return err
	}
	schedules := strings.Split(cfg[`schedules`], "|")
	scheduler := cron.New()
	for _, schedule := range schedules {
		scheduler.AddFunc(schedule, func() {
			if err := RunJob(ctx, db, msgChan); err != nil {
				fmt.Println(err.Error())
			}
		})
	}
	scheduler.Start()
	defer scheduler.Stop()
	return nil
}
