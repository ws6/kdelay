package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/beego/beego/v2/core/config"
	"github.com/ws6/klib"
	"github.com/ws6/msi"
)

const (
	RELEASE_AT = `kdelay_release_at`
)

func main() {
	if err := mainErr(); err != nil {
		fmt.Println(err.Error())
	}
}

func mainErr() error {

	cfgr, err := config.NewConfig(`ini`, os.Getenv(`KDELAY_CONFIG_FILE`))
	if err != nil {

		return err
	}
	dbCfg, err := cfgr.GetSection(`mysql_config`)
	if err != nil {
		return err
	}
	dbConnStr := getMySqlConfig(dbCfg)
	db, err := msi.NewDb(msi.MYSQL, dbConnStr, dbCfg[`db`], ``)
	if err != nil {
		return err
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	go consume(ctx, db, cfgr)
	//TODO produce to chan
	ch, err := produceChan(ctx, cfgr)
	if err != nil {
		return err
	}
	defer close(ch)
	go RunSchedules(ctx, db, cfgr, ch)
	return nil
}

func getMySqlConfig(cfg map[string]string) string {
	return fmt.Sprintf(`%s:%s@(%s:%s)/%s`,
		cfg[`user`],
		cfg[`pass`],
		cfg[`host`],
		cfg[`port`],
		cfg[`db`],
	)
}

func kmsgToDb(ctx context.Context, db *msi.Msi, msg *klib.Message) error {
	if msg == nil || msg.Headers == nil {
		return fmt.Errorf(`bad message`)
	}
	messageTable := db.GetTable(`message`)
	if messageTable == nil {
		return fmt.Errorf(`no message table`)
	}

	releaseAtStr, ok := msg.Headers[RELEASE_AT]
	if !ok {
		return fmt.Errorf(`no %s in header`, RELEASE_AT)
	}
	releaseAtUnix, err := strconv.ParseInt(releaseAtStr, 10, 64)
	if err != nil {
		return fmt.Errorf(`strconv:%s`, err.Error())
	}
	t := time.Unix(releaseAtUnix, 0)
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return messageTable.Insert(
		msi.M{
			`release_at`: t,
			`body`:       string(body),
		},
	)

}

func consume(ctx context.Context, db *msi.Msi, cfg config.Configer) error {

	consumerCfg, err := cfg.GetSection(`consumer_config`)
	if err != nil {
		return err
	}
	consumer, err := klib.NewKlib(consumerCfg)
	if err != nil {
		return err
	}

	//TODO new producer

	//TODO new db connection

	//TODO add crons

	consumer.ConsumeLoop(ctx, consumerCfg[`kdelay_topic`], func(msg *klib.Message) error {
		//TODO store it with corrected release_at
		return kmsgToDb(ctx, db, msg)
	})
	return nil
}

func produceChan(ctx context.Context, cfg config.Configer) (chan *klib.Message, error) {

	producerCfg, err := cfg.GetSection(`producer_config`)
	if err != nil {
		return nil, err
	}

	k, err := klib.NewKlib(producerCfg)
	if err != nil {
		return nil, err
	}
	chanSize := 100
	if s, ok := producerCfg[`chan_size`]; ok {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			chanSize = n
		}
	}
	ret := make(chan *klib.Message, chanSize)
	go func() {
		k.ProduceChan(ctx, producerCfg[`producer_topic`], ret)
	}()

	return ret, nil

}
