package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli"
)

type Action string

const (
	WITHDRAW Action = "withdraw"
	DEPOSIT  Action = "deposit"
)

var (
	DatabasePath = filepath.Join(os.Getenv("HOME"), ".trdb")
)

type Value int

func (v Value) String() string {
	return fmt.Sprintf("%d,%02d€", v/100, v%100)
}

func (v Value) Add(a Value) Value {
	return v + a
}

func Parse(in string) Value {
	var maj, min int
	fmt.Sscanf(in, "%d,%02d", &maj, &min)
	return Value(maj*100 + min%100)
}

type Transaction struct {
	Name   string    `json:"name"`
	Amount Value     `json:"amount"`
	Type   Action    `json:"type"`
	Date   time.Time `json:"date"`
}

func NewTransaction(name string, action Action, amount Value) Transaction {
	return Transaction{
		Name:   name,
		Amount: amount,
		Type:   action,
		Date:   time.Now(),
	}
}

type Database struct {
	Name         string        `json:"name"`
	Transactions []Transaction `json:"transaction"`
}

func NewDatabase(name string) Database {
	return Database{
		Name:         name,
		Transactions: make([]Transaction, 0),
	}
}

func (db *Database) Store(transact Transaction) {
	db.Transactions = append(db.Transactions, transact)
}

func openDatabase() (Database, error) {
	var database Database

	bytes, err := ioutil.ReadFile(DatabasePath)
	if err != nil {
		return Database{}, err
	}
	err = json.Unmarshal(bytes, &database)
	if err != nil {
		return Database{}, nil
	}
	return database, nil
}

func writeDatabase(database Database) error {
	json, err := json.Marshal(database)
	if err != nil {
		return err
	}
	ioutil.WriteFile(DatabasePath, json, 0644)
	return nil
}

func storeTransaction(transact Transaction) error {
	database, err := openDatabase()
	if err != nil {
		return err
	}
	database.Store(transact)
	err = writeDatabase(database)
	return err
}

func main() {
	app := cli.NewApp()
	app.Name = "transaction"

	app.Commands = []cli.Command{
		{
			Name:  "init",
			Usage: "Initialize the database",
			Action: func(c *cli.Context) error {
				fmt.Print("Database name: ")
				var name string
				fmt.Scanf("%s\n", &name)
				database := NewDatabase(name)
				err := writeDatabase(database)
				if err != nil {
					return err
				}
				fmt.Printf("Created the database '%s'.\n", name)
				return nil
			},
		},
		{
			Name:  "store",
			Usage: "Store a new transaction",
			Action: func(c *cli.Context) error {
				var name string
				for name == "" {
					fmt.Print("Transaction name: ")
					fmt.Scanln(&name)
					name = strings.TrimSpace(name)
				}

				var action Action
				for action == "" {
					fmt.Print("Transaction type (wd / dp): ")
					fmt.Scanln(&action)
					action = Action(strings.TrimSpace(string(action)))
					if action == "wd" {
						action = WITHDRAW
					} else if action == "dp" {
						action = DEPOSIT
					} else {
						action = ""
					}
				}

				var amountString string
				var amount Value
				for amount == 0 {
					fmt.Print("Transaction amount: ")
					fmt.Scanln(&amountString)
					amount = Parse(amountString)
				}

				transact := NewTransaction(name, action, amount)
				err := storeTransaction(transact)
				if err != nil {
					return err
				}

				fmt.Printf("Stored the %s transaction '%s' (%s)\n", action, name, amount.String())

				return nil
			},
		},
		{
			Name:  "list",
			Usage: "List all transactions",
			Action: func(c *cli.Context) error {
				db, err := openDatabase()
				if err != nil {
					return err
				}

				fmt.Printf("%20s  ", db.Name)
				for i := 0; i < 46; i++ {
					fmt.Print("=")
				}
				fmt.Println()

				var balance Value
				for _, transact := range db.Transactions {
					fmt.Printf("On %16s %16s :: %-8s %12s\n", formatTime(transact.Date), transact.Name, transact.Type, transact.Amount)

					switch transact.Type {
					case WITHDRAW:
						balance = balance.Add(-transact.Amount)
					case DEPOSIT:
						balance = balance.Add(transact.Amount)
					}
				}

				fmt.Printf("%56s------------\n%56s%12s\n", "", "", balance)
				return nil
			},
		},
	}

	app.Run(os.Args)
}

func formatTime(t time.Time) string {
	return fmt.Sprintf("%02d. %s %04d %02d:%02d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
}
