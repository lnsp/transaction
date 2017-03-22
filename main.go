package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lnsp/transaction/db"
	"github.com/metakeule/fmtdate"
	"github.com/urfave/cli"
)

const (
	// HeaderSymbol used for displaying table hreaders.
	tableHeaderSymbol = "="
	// TimeFormat to display transaction timestamps.
	transactionTimeFormat = "%02d. %s %04d %02d:%02d"

	abortedMessage           = "Action aborted."
	wipeDatabaseConfirmation = "A database already exists. Are you sure you want to do this? (y / N): "
	wipeDatabaseYes          = "y"
	wipeDatabaseNo           = "n"

	databaseNameField      = "Database name: "
	createdDatabaseMessage = "Created the database '%s'.\n"

	transactionNameField      = "Transaction name: "
	transactionTypeField      = "Transaction type (wd / dp): "
	transactionDateField      = "Transaction date: "
	transactionDateFormat     = "D.M.YYYY"
	transactionTypeWithdraw   = "wd"
	transactionTypeDeposit    = "dp"
	transactionAmountField    = "Transaction amount: "
	transactionSuccessMessage = "Stored the %s transaction '%s' (%s).\n"

	wipeTransactionYes          = "y"
	wipeTransactionNo           = "n"
	wipeTransactionConfirmation = "\nAre you sure? (y / N) "
	wipeTransactionSuccess      = "Transaction deleted."
)

var (
	console = bufio.NewReader(os.Stdin)
)

func isTypeDeposit(text string) bool {
	lc := strings.ToLower(strings.TrimSpace(text))
	return lc == "dp" || lc == "deposit" || lc == "depo"
}

func isTypeWithdraw(text string) bool {
	lc := strings.ToLower(strings.TrimSpace(text))
	return lc == "wd" || lc == "withdraw" || lc == "draw"
}

func initAction(c *cli.Context) error {
	if db.Exists() && !c.Bool("force") {
		fmt.Print(wipeDatabaseConfirmation)
		status := wipeDatabaseNo
		fmt.Scanf("%s")
		if status != wipeDatabaseYes {
			fmt.Println(abortedMessage)
			return nil
		}
	}
	fmt.Print(databaseNameField)
	name, _ := getInput()
	database := db.NewDatabase(name)
	err := db.Write(database)
	if err != nil {
		return err
	}
	fmt.Printf(createdDatabaseMessage, name)
	return nil
}

func storeAction(c *cli.Context) error {
	var name string
	for name == "" {
		fmt.Print(transactionNameField)
		name, _ = getInput()
	}
	var date time.Time
	fmt.Print(transactionDateField)
	dateStr, _ := getInput()
	date, err := fmtdate.Parse(transactionDateFormat, dateStr)
	if err != nil {
		date = time.Now()
	}
	var action db.Action
	for action == "" {
		fmt.Print(transactionTypeField)
		actionString, _ := getInput()
		if isTypeWithdraw(actionString) {
			action = db.Withdraw
		} else if isTypeDeposit(actionString) {
			action = db.Deposit
		} else {
			action = ""
		}
	}
	var amount db.Value
	for amount == 0 {
		fmt.Print(transactionAmountField)
		amountString, _ := getInput()
		amount = db.Parse(amountString)
	}
	transact := db.NewTransaction(name, action, amount, date)
	err = db.Store(transact)
	if err != nil {
		return err
	}
	fmt.Printf(transactionSuccessMessage, action, name, amount.String())
	return nil
}

func limitString(s string, l int) string {
	if len(s) < l {
		return fmt.Sprintf("%"+strconv.Itoa(l)+"s", s)
	}
	return string([]rune(s)[:l])
}

func getTableHeader(headerText string) string {
	header := headerText + "  "
	for i := 0; i < 46; i++ {
		header += tableHeaderSymbol
	}
	return limitString(header, 82)
}

func printTransactionTable(header string, transactions map[int]db.Transaction) {
	fmt.Println(getTableHeader(header))
	var ids []int
	for i := range transactions {
		ids = append(ids, i)
	}
	sort.Ints(ids)
	var balance db.Value
	for _, id := range ids {
		transact := transactions[id]
		idString := "[#" + strconv.Itoa(id) + "]"
		fmt.Printf("%6s  On %s %s :: %-8s %12s\n", idString, limitString(formatTime(transact.Date), 24), limitString(transact.Name, 20), transact.Type, transact.Amount)

		switch transact.Type {
		case db.Withdraw:
			balance = balance.Add(-transact.Amount)
		case db.Deposit:
			balance = balance.Add(transact.Amount)
		}
	}
	fmt.Printf("%69s------------\n%69s%12s\n", "", "", balance)
}

func listAction(c *cli.Context) error {
	database, err := db.Open()
	if err != nil {
		return err
	}
	idMap := make(map[int]db.Transaction)
	startValue := database.Size() - c.Int("limit")
	for id := database.Size() - 1; id >= 0 && id >= startValue; id-- {
		transact, err := database.Read(id)
		if err != nil {
			return err
		}
		idMap[id] = transact
	}
	header := fmt.Sprintf("%s (latest %d entries)", database.Name, len(idMap))
	printTransactionTable(header, idMap)
	return nil
}

func filterAction(c *cli.Context) error {
	database, err := db.Open()
	if err != nil {
		return err
	}
	namePredicate, maxPredicate, minPredicate, typePredicate := c.String("name"), db.Parse(c.String("max")), db.Parse(c.String("min")), c.String("type")
	header := fmt.Sprintf("%s (name='%s', min='%s', max='%s', type='%s')", database.Name, namePredicate, minPredicate, maxPredicate, typePredicate)
	idMap := make(map[int]db.Transaction)
	for id := 0; id < database.Size(); id++ {
		transact, err := database.Read(id)
		if err != nil {
			return err
		}
		if namePredicate != "" && transact.Name != namePredicate {
			continue
		}
		if maxPredicate != db.ZeroValue && maxPredicate.Smaller(transact.Amount) {
			continue
		}
		if minPredicate != db.ZeroValue && minPredicate.Larger(transact.Amount) {
			continue
		}
		if typePredicate != "" && ((isTypeDeposit(typePredicate) && transact.Type != db.Deposit) || (isTypeWithdraw(typePredicate) && transact.Type != db.Withdraw)) {
			continue
		}
		idMap[id] = transact
	}
	printTransactionTable(header, idMap)
	return nil
}

func deleteAction(c *cli.Context) error {
	ID, err := strconv.Atoi(c.Args().First())
	if err != nil {
		return err
	}
	transaction, err := db.Get(ID)
	if err != nil {
		return err
	}
	fmt.Print(transaction, wipeTransactionConfirmation)
	confirmation, err := getInput()
	if err != nil {
		return err
	}
	if confirmation != wipeTransactionYes {
		fmt.Println(abortedMessage)
		return nil
	}
	err = db.Delete(ID)
	if err != nil {
		return err
	}
	fmt.Println(wipeTransactionSuccess)
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "transaction"
	app.Authors = []cli.Author{
		{Name: "Lennart Espe", Email: "lennart@espe.tech"},
	}
	app.Copyright = "(c) 2016 Lennart Espe"
	app.Usage = "A housekeeping book in your terminal."
	app.Version = "0.2"
	app.Commands = []cli.Command{
		{
			Name:   "init",
			Usage:  "Initialize the database",
			Action: initAction,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "force, f",
					Usage: "Disable any warnings",
				},
			},
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
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "limit, l",
					Value: 10,
					Usage: "Amount of entries shown",
				},
			},
		},
		{
			Name:   "delete",
			Usage:  "Delete a transaction",
			Action: deleteAction,
		},
		{
			Name:   "filter",
			Usage:  "Filter and list matching transactions",
			Action: filterAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name",
					Value: "",
					Usage: "Filter by name (case sensitive)",
				},
				cli.StringFlag{
					Name:  "min",
					Value: "",
					Usage: "Filter by minimum volume (in standard currency format)",
				},
				cli.StringFlag{
					Name:  "max",
					Value: "",
					Usage: "Filter by maximum volume (in standard currency format)",
				},
				cli.StringFlag{
					Name:  "type",
					Value: "",
					Usage: "Filter transaction by type (withdraw or deposit)",
				},
			},
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
	return fmt.Sprintf(transactionTimeFormat, t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
}

func validIndex(x, max int) bool {
	return x >= 0 && x <= max
}
