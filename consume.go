package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"os/signal"
	"syscall"

	"github.com/beego/beego/v2/core/config"
	_ "github.com/go-sql-driver/mysql"
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

func getTableName(cfg config.Configer) (string, error) {
	return cfg.String(`kdelay::tablename`)
}

var GetTableName, SetTableName = func() (
	func() string,
	func(string),
) {
	var _table_name = `message`
	return func() string {
			return _table_name
		},
		func(n string) {
			_table_name = n
		}

}()

func getExcutablePath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return exPath
}

func getDefaultConfigPath() string {
	exePath := getExcutablePath()
	return filepath.Join(exePath, `conf`, `app.conf`)
}

func mainErr() error {
	configPath := getDefaultConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = os.Getenv(`KDELAY_CONFIG_FILE`)
	}

	cfgr, err := config.NewConfig(`ini`, configPath)
	if err != nil {
		return err
	}
	tableName, err := getTableName(cfgr)
	if err != nil {
		return err
	}
	SetTableName(tableName)

	dbCfg, err := cfgr.GetSection(`mysql_config`)
	if err != nil {
		return err
	}
	dbConnStr := getMySqlConfig(dbCfg)

	//create table if not exists
	db0, err := msi.NewDb(msi.MYSQL, dbConnStr, dbCfg[`db`], ``)
	if err != nil {
		return err
	}
	if err := createTable(db0, GetTableName()); err != nil {
		return fmt.Errorf(`createTable:%s`, err.Error())
	}
	db0.Close()

	db, err := msi.NewDb(msi.MYSQL, dbConnStr, dbCfg[`db`], ``)
	if err != nil {
		return err
	}
	if n, err := cfgr.Int(`mysql_config::max_connection`); err == nil && n > 0 {
		db.Db.SetMaxOpenConns(n)
	}

	defer db.Close()

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigchan:
			cancelFn()
			fmt.Println(`canceld from Signal`)
		}
	}()
	go consume(ctx, db, cfgr)
	//TODO produce to chan
	ch, err := produceChan(ctx, cfgr)
	if err != nil {
		return err
	}
	defer close(ch)
	return RunSchedules(ctx, db, cfgr, ch)

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
	messageTable := db.GetTable(GetTableName())
	if messageTable == nil {
		return fmt.Errorf(`no message table`)
	}

	releaseAtStr, ok := msg.Headers[RELEASE_AT]
	if !ok {
		return fmt.Errorf(`no %s in header`, RELEASE_AT)
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	releaseAt := TryParseTime(releaseAtStr)
	fmt.Println(`got a message want to release at`, releaseAt)
	return messageTable.Insert(
		msi.M{
			`release_at`: releaseAt,
			`body`:       string(body),
		},
	)

}

func consume(ctx context.Context, db *msi.Msi, cfg config.Configer) error {
	fmt.Println(`consumer started`)
	consumerCfg, err := cfg.GetSection(`consumer_config`)
	if err != nil {
		return err
	}
	consumer, err := klib.NewKlib(consumerCfg)
	if err != nil {
		return err
	}

	consumer.ConsumeLoop(ctx, consumerCfg[`consumer_topic`], func(msg *klib.Message) error {
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
		fmt.Println(`producer started`)
		k.ProduceChan(ctx, producerCfg[`producer_topic`], ret)
	}()

	return ret, nil

}
