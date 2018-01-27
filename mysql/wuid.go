package wuid

import (
	"database/sql"
	"errors"
	"log"
	"sync/atomic"

	"github.com/edwingeng/wuid/internal"
	_ "github.com/go-sql-driver/mysql"
)

type WUID struct {
	w *internal.WUID
}

func NewWUID(tag string, logger *log.Logger) *WUID {
	return &WUID{w: internal.NewWUID(tag, logger)}
}

func (this *WUID) Next() uint64 {
	return this.w.Next()
}

func (this *WUID) LoadH24FromMysql(addr, user, pass, dbName, table string) error {
	if len(addr) == 0 {
		return errors.New("addr cannot be empty")
	}
	if len(user) == 0 {
		return errors.New("user cannot be empty")
	}
	if len(dbName) == 0 {
		return errors.New("dbName cannot be empty")
	}
	if len(table) == 0 {
		return errors.New("table cannot be empty")
	}

	var dsn string
	dsn += user
	if len(pass) > 0 {
		dsn += ":" + pass
	}
	dsn += "@tcp(" + addr + ")/" + dbName

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec("REPLACE INTO wuid (x) VALUES (0)")
	if err != nil {
		return err
	}
	lastInsertedId, err := result.LastInsertId()
	if err != nil {
		return err
	}

	atomic.StoreUint64(&this.w.N, uint64(lastInsertedId&0x0FFF)<<40)

	this.w.Lock()
	defer this.w.Unlock()

	if this.w.Renew != nil {
		return nil
	}
	this.w.Renew = func() error {
		return this.LoadH24FromMysql(addr, user, pass, dbName, table)
	}

	return nil
}
