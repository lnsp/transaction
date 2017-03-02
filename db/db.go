package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const (
	// The default database suffix.
	defaultDatabaseSuffix = ".trdb"
)

var (
	// Transaction could not be found (maybe invalid ID?)
	errTransactionNotFound = errors.New("not found: the transaction does not exist")
	// The default database storage path.
	defaultDatabasePath = filepath.Join(os.Getenv("HOME"), defaultDatabaseSuffix)
)

// Currency stores information about a currency.
type Currency struct {
	Name, Format string
	Ratio        Value
}

var (
	// Euro currency
	Euro = Currency{"Euro", "%d.%02d€", Value(100)}
	// Dollar currency
	Dollar = Currency{"Dollar", "%d.%02d$", Value(100)}
	// DefaultCurrency for display
	DefaultCurrency = Euro
)

// Action is a transaction type.
type Action string

const (
	// Withdraw takes money from the account.
	Withdraw Action = "withdraw"
	// Deposit stores money on the account.
	Deposit Action = "deposit"
)

// Value is a specific amount of money.
type Value int

const (
	// ZeroValue represents a 0.
	ZeroValue = Value(0)
)

func abs(x Value) Value {
	if x < ZeroValue {
		return -x
	}
	return x
}

// Stringifies the value in a currency format.
func (v Value) String() string {
	return fmt.Sprintf(DefaultCurrency.Format, v/DefaultCurrency.Ratio, abs(v%DefaultCurrency.Ratio))
}

// Add more money onto the existing value.
func (v Value) Add(a Value) Value {
	return v + a
}

// Smaller compares if the value is smaller than the argument.
func (v Value) Smaller(a Value) bool {
	return int(v) < int(a)
}

// Larger compares if the value is larger than the argument.
func (v Value) Larger(a Value) bool {
	return int(v) > int(a)
}

// Parse a string into a pile of money.
func Parse(in string) Value {
	var maj, min int
	fmt.Sscanf(in, DefaultCurrency.Format, &maj, &min)
	return Value(Value(maj)*DefaultCurrency.Ratio + Value(min)%DefaultCurrency.Ratio)
}

// Transaction stores a virtual transaction.
type Transaction struct {
	Name   string    `json:"name"`
	Amount Value     `json:"amount"`
	Type   Action    `json:"type"`
	Date   time.Time `json:"date"`
}

// NewTransaction initializes a new transaction.
func NewTransaction(name string, action Action, amount Value) Transaction {
	return Transaction{
		Name:   name,
		Amount: amount,
		Type:   action,
		Date:   time.Now(),
	}
}

// Database with a name and a list of transactions.
type Database struct {
	Name         string        `json:"name"`
	Transactions []Transaction `json:"transaction"`
}

// NewDatabase intializes a empty list of transactions.
func NewDatabase(name string) Database {
	return Database{
		Name:         name,
		Transactions: make([]Transaction, 0),
	}
}

// Size returns the count of transactions.
func (db *Database) Size() int {
	return len(db.Transactions)
}

// Store the transaction in the database.
func (db *Database) Store(transact Transaction) {
	db.Transactions = append(db.Transactions, transact)
}

// Delete a transaction at the given position.
func (db *Database) Delete(ID int) error {
	if ID < 0 || ID >= db.Size() {
		return errTransactionNotFound
	}
	db.Transactions = append(db.Transactions[:ID], db.Transactions[ID+1:]...)
	return nil
}

// Retrieve a transaction from the database.
func (db *Database) Read(ID int) (Transaction, error) {
	if ID < 0 || ID >= db.Size() {
		return Transaction{}, errTransactionNotFound
	}
	return db.Transactions[ID], nil
}

// Open a existing database.
func Open() (Database, error) {
	var database Database

	bytes, err := ioutil.ReadFile(defaultDatabasePath)
	if err != nil {
		return Database{}, err
	}
	err = json.Unmarshal(bytes, &database)
	if err != nil {
		return Database{}, nil
	}
	return database, nil
}

// Exists is true if the database already exists.
func Exists() bool {
	if _, err := os.Stat(defaultDatabasePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// Write the database to the hard drive.
func Write(database Database) error {
	json, err := json.Marshal(database)
	if err != nil {
		return err
	}
	ioutil.WriteFile(defaultDatabasePath, json, 0644)
	return nil
}

// Store the transaction in the existing database.
func Store(transact Transaction) error {
	database, err := Open()
	if err != nil {
		return err
	}
	database.Store(transact)
	err = Write(database)
	return err
}

// Get a transaction from an existing database.
func Get(ID int) (Transaction, error) {
	database, err := Open()
	if err != nil {
		return Transaction{}, err
	}
	return database.Read(ID)
}

// Delete a transaction from an existing database.
func Delete(ID int) error {
	database, err := Open()
	if err != nil {
		return err
	}
	err = database.Delete(ID)
	if err != nil {
		return err
	}
	err = Write(database)
	return err
}
