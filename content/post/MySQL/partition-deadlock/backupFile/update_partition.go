package main

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"fmt"
	"sync"
	"time"
)

const (
	dbHost         string = "10.17.117.203"
	dbPort         int    = 3331
	dbUser         string = "root"
	dbPassword     string = "root"
	dbDatabase     string = "test"
	dbMaxOpenConns int    = 10
	dbMaxIdleConns int    = 10
	dbMaxLifetime  int    = 3600
)

var DB *sql.DB

func initDB() {
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbUser, dbPassword, dbHost, dbPort, dbDatabase)
	DB, err = sql.Open("mysql", dsn)

	DB.SetMaxOpenConns(dbMaxOpenConns)
	DB.SetMaxIdleConns(dbMaxIdleConns)
	DB.SetConnMaxLifetime(time.Duration(dbMaxLifetime) * time.Second)

	if err != nil {
		fmt.Println("connection to mysql failed:", err)
		return
	}
	fmt.Println("connnect success")
}

func txUpdateTest(roundId string, endTime string, wg *sync.WaitGroup) {
	defer wg.Done()

	// begin transaction
	tx, err := DB.Begin()
	if err != nil {
		fmt.Println("begin fail: ", err)
		return
	}

	// prepare
	stmt, err := tx.Prepare("UPDATE `round_to_txn` SET `end_time` = ? WHERE `round_id` = ?") // 會 deadlock
	// stmt, err := tx.Prepare("UPDATE `round_to_txn` SET `end_time` = ? WHERE `round_id` = ? AND `end_time` = '0000-00-00 00:00:00'") // 不會 deadlock
	// stmt, err := tx.Prepare("UPDATE `round_to_txn` SET `end_time` = ? WHERE `round_id` = ? AND `end_time` <= '2022-10-31 23:59:59'") // 不會 deadlock
	//stmt, err := tx.Prepare("UPDATE `round_to_txn` SET `end_time` = ? WHERE `round_id` = ? AND (`end_time` <= '2022-10-31 23:59:59' OR `end_time` >= '2022-12-01 00:00:00' )") // 不會 deadlock
	// stmt, err := tx.Prepare("UPDATE `round_to_txn` SET `end_time` = ? WHERE `round_id` = ? AND (`end_time` <= '2022-11-10 23:59:59')") // 會 deadlock
	if err != nil {
		fmt.Println("prepare fail: ", err)
		return
	}

	// exec
	_, err = stmt.Exec(endTime, roundId)
	if err != nil {
		fmt.Println("Exec fail: ", err)
		return
	}

	// commit
	tx.Commit()
}

func updateTest(roundId string, endTime string) {
	_, err := DB.Exec("UPDATE `round_to_txn` SET `end_time` = ? WHERE `round_id` = ?", endTime, roundId)
	if err != nil {
		fmt.Println("update fail:", err)
		return
	}
}

func main() {
	wg := new(sync.WaitGroup)
	initDB()
	defer DB.Close()

	type sqlValue struct {
		roundId string
		endTime string
	}

	var initValue = []sqlValue{}
	for i := 10; i < 21; i++ {
		initValue = append(initValue, sqlValue{fmt.Sprintf("0399%deukXEC", i), "0000-00-00 00:00:00"})
	}

	for _, e := range initValue {
		updateTest(e.roundId, e.endTime)
	}

	var testValue = []sqlValue{}
	for i := 10; i < 21; i++ {
		testValue = append(testValue, sqlValue{fmt.Sprintf("0399%deukXEC", i), "2022-11-16 08:53:08"})
	}

	for _, e := range testValue {
		wg.Add(1)
		go txUpdateTest(e.roundId, e.endTime, wg)
	}

	wg.Wait()
}