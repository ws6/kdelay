package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/config"

	"github.com/robfig/cron/v3"
	"github.com/ws6/dlock"
	"github.com/ws6/klib"
	"github.com/ws6/msi"
)

func init() {
	// msi.DEBUG = true
}
func timeToSQLTimeStr(t time.Time) string {
	return fmt.Sprintf(`%04d-%02d-%02d %02d:%02d:%02d`, t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func makeQuery() msi.M {

	return msi.M{
		`release_at`: map[string]interface{}{
			msi.LTE: timeToSQLTimeStr(time.Now()),
		},
	}
}

func getLimit() int {
	return 100
}

//scan db and then run the delay publish
func RunJob(ctx context.Context, db *msi.Msi, msgChan chan *klib.Message) error {
	fmt.Println(`cron wake up`)
	messageTable := db.GetTable(GetTableName())
	if messageTable == nil {
		return fmt.Errorf(`no [%s] table`, GetTableName())
	}
	c := messageTable.Find(
		makeQuery(),
		msi.M{
			msi.ORDERBY: []string{`id`},
		},
	).CtxChan(ctx, getLimit())
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
		fmt.Println(`delay produced a message`, kmsg)
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
	dlockEnabled, _ := cfgr.Bool(`dlock_config::enabled`)
	dlockCfg := make(map[string]string)
	if dlockEnabled {
		fmt.Println(`dlock enabled`)
		_dlockCfg, err := cfgr.GetSection(`dlock_config`)
		if err != nil {
			return err
		}

		dlockCfg = _dlockCfg
	}
	for _, schedule := range schedules {
		fmt.Println(`start a scheduler`, schedule)
		scheduler.AddFunc(schedule, func() {

			//aquire dlock to run
			if dlockEnabled {
				lock, err := dlock.NewDlock(dlockCfg)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				getDlockKey := func() string {
					return fmt.Sprintf(`%s`, dlockCfg[`dlock_key`])
				}
				dm := lock.NewMutex(ctx, getDlockKey())

				if err := dm.Lock(); err != nil {
					fmt.Println(`another thread locked`, err.Error())
					return
				}
				defer dm.Unlock()
				fmt.Println(`acquired a lock to run`)
			}

			if err := RunJob(ctx, db, msgChan); err != nil {
				fmt.Println(err.Error())
			}
		})
	}
	scheduler.Start()
	defer scheduler.Stop()

	select {
	case <-ctx.Done():
		fmt.Println(`canceled`)
		return ctx.Err()
	}
	return nil
}
