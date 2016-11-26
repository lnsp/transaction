package main

import (
	"bufio"
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

const (
	CURRENCY_FORMAT = "%d.%02d€"
	MAJ_MIN         = 100
	HEADER_SYMBOL   = "="
	TIME_FORMAT     = "%02d. %s %04d %02d:%02d"
	DB_NAME         = ".trdb"
)

var (
	DatabasePath = filepath.Join(os.Getenv("HOME"), DB_NAME)
)

type Value int

func (v Value) String() string {
	return fmt.Sprintf(CURRENCY_FORMAT, v/MAJ_MIN, v%MAJ_MIN)
}

func (v Value) Add(a Value) Value {
	return v + a
}

func Parse(in string) Value {
	var maj, min int
	fmt.Sscanf(in, CURRENCY_FORMAT, &maj, &min)
	return Value(maj*MAJ_MIN + min%MAJ_MIN)
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

func databaseExists() bool {
	if _, err := os.Stat(DatabasePath); os.IsNotExist(err) {
		return false
	}
	return true
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

	console := bufio.NewReader(os.Stdin)

	app.Commands = []cli.Command{
		{
			Name:  "init",
			Usage: "Initialize the database",
			Action: func(c *cli.Context) error {
				if databaseExists() {
					fmt.Print("A database already exists. Are you sure you want to do this? (y / N): ")
					status := "n"
					fmt.Scanf("%s")
					if status != "y" {
						fmt.Println("Action aborted.")
						return nil
					}
				}

				fmt.Print("Database name: ")
				name, _ := console.ReadString('\n')
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
					name, _ := console.ReadString('\n')
					name = strings.TrimSpace(name)
				}

				var action Action
				for action == "" {
					fmt.Print("Transaction type (wd / dp): ")
					actionString, _ := console.ReadString('\n')
					action = Action(strings.TrimSpace(actionString))
					if action == "wd" {
						action = WITHDRAW
					} else if action == "dp" {
						action = DEPOSIT
					} else {
						action = ""
					}
				}

				var amount Value
				for amount == 0 {
					fmt.Print("Transaction amount: ")
					amountString, _ := console.ReadString('\n')
					amount = Parse(amountString)
				}

				transact := NewTransaction(name, action, amount)
				err := storeTransaction(transact)
				if err != nil {
					return err
				}

				fmt.Printf("Stored the %s transaction '%s' (%s).\n", action, name, amount.String())

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
	return fmt.Sprintf(TIME_FORMAT, t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
}
