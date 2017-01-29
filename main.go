package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lnsp/transaction/db"
	"github.com/urfave/cli"
)

const (
	// The used header symbol.
	HEADER_SYMBOL = "="
	// The time format used in transaction listing.
	TIME_FORMAT = "%02d. %s %04d %02d:%02d"
)

var (
	console = bufio.NewReader(os.Stdin)
)

func initAction(c *cli.Context) error {
	if db.Exists() {
		fmt.Print("A database already exists. Are you sure you want to do this? (y / N): ")
		status := "n"
		fmt.Scanf("%s")
		if status != "y" {
			fmt.Println("Action aborted.")
			return nil
		}
	}

	fmt.Print("Database name: ")
	name, _ := getInput()
	database := db.NewDatabase(name)
	err := db.Write(database)
	if err != nil {
		return err
	}

	fmt.Printf("Created the database '%s'.\n", name)
	return nil
}

func storeAction(c *cli.Context) error {
	var name string
	for name == "" {
		fmt.Print("Transaction name: ")
		name, _ = getInput()
	}

	var action db.Action
	for action == "" {
		fmt.Print("Transaction type (wd / dp): ")
		actionString, _ := getInput()
		action = db.Action(actionString)
		if action == "wd" {
			action = db.WITHDRAW
		} else if action == "dp" {
			action = db.DEPOSIT
		} else {
			action = ""
		}
	}

	var amount db.Value
	for amount == 0 {
		fmt.Print("Transaction amount: ")
		amountString, _ := getInput()
		amount = db.Parse(amountString)
	}

	transact := db.NewTransaction(name, action, amount)
	err := db.Store(transact)
	if err != nil {
		return err
	}

	fmt.Printf("Stored the %s transaction '%s' (%s).\n", action, name, amount.String())

	return nil
}

func limitString(s string, l int) string {
	if len(s) < l {
		return fmt.Sprintf("%"+strconv.Itoa(l)+"s", s)
	}
	return string([]rune(s)[:l])
}

func listAction(c *cli.Context) error {
	database, err := db.Open()
	if err != nil {
		return err
	}

	fmt.Printf("%33s  ", database.Name)
	for i := 0; i < 46; i++ {
		fmt.Print("=")
	}
	fmt.Println()

	var balance db.Value
	for ID := 0; ID < database.Size(); ID++ {
		transact, err := database.Read(ID)
		if err != nil {
			return err
		}
		idString := "[#" + strconv.Itoa(ID) + "]"
		fmt.Printf("%6s  On %s %s :: %-8s %12s\n", idString, limitString(formatTime(transact.Date), 24), limitString(transact.Name, 20), transact.Type, transact.Amount)

		switch transact.Type {
		case db.WITHDRAW:
			balance = balance.Add(-transact.Amount)
		case db.DEPOSIT:
			balance = balance.Add(transact.Amount)
		}
	}

	fmt.Printf("%69s------------\n%69s%12s\n", "", "", balance)
	return nil
}

func deleteAction(c *cli.Context) error {
	ID, err := strconv.Atoi(c.Args().First())
	if err != nil {
		return db.TransactionNotFound
	}

	transaction, err := db.Get(ID)
	if err != nil {
		return err
	}

	fmt.Print(transaction, "\nAre you sure? (y / N) ")
	confirmation, err := getInput()
	if err != nil {
		return err
	}

	if confirmation != "y" {
		fmt.Println("Action aborted.")
		return nil
	}

	err = db.Delete(ID)
	if err != nil {
		return err
	}

	fmt.Println("Transaction deleted.")
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "transaction"
	app.Authors = []cli.Author{
		{"Lennart Espe", "lennart@espe.tech"},
	}
	app.Copyright = "(c) 2016 Lennart Espe"
	app.Usage = "A housekeeping book in your terminal."
	app.Version = "0.1"

	app.Commands = []cli.Command{
		{
			Name:   "init",
			Usage:  "Initialize the database",
			Action: initAction,
		},
		{
			Name:   "store",
			Usage:  "Store a new transaction",
			Action: storeAction,
		},
		{
			Name:   "list",
			Usage:  "List all transactions",
			Action: listAction,
		},
		{
			Name:   "delete",
			Usage:  "Delete a transaction",
			Action: deleteAction,
		},
	}

	app.Run(os.Args)
}

func getInput() (string, error) {
	input, err := console.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func formatTime(t time.Time) string {
	return fmt.Sprintf(TIME_FORMAT, t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
}

func validIndex(x, max int) bool {
	return x >= 0 && x <= max
}
