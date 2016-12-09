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
	DB_NAME = ".trdb"
)

var (
	// Transaction could not be found (maybe invalid ID?)
	TransactionNotFound = errors.New("Not found: The transaction does not exist.")
	// The default database storage path.
	DatabasePath = filepath.Join(os.Getenv("HOME"), DB_NAME)
)

type Currency struct {
	Name, Format string
	Ratio        Value
}

var (
	EURO            Currency = Currency{"Euro", "%d.%02d€", Value(100)}
	DefaultCurrency          = EURO
)

type Action string

const (
	// A withdrawal, taking money from the account.
	WITHDRAW Action = "withdraw"
	// A deposit, storing money on the account.
	DEPOSIT Action = "deposit"
)

// Value is a specific amount of money.
type Value int

// Stringifies the value in a currency format.
func (v Value) String() string {
	return fmt.Sprintf(DefaultCurrency.Format, v/DefaultCurrency.Ratio, v%DefaultCurrency.Ratio)
}

// Adds more money onto the existing value.
func (v Value) Add(a Value) Value {
	return v + a
}

// Parse a string into a pile of money.
func Parse(in string) Value {
	var maj, min int
	fmt.Sscanf(in, DefaultCurrency.Format, &maj, &min)
	return Value(Value(maj)*DefaultCurrency.Ratio + Value(min)%DefaultCurrency.Ratio)
}

// A virtual transaction.
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

// A database with a name and a list of transactions.
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

// Stores the transaction in the database.
func (db *Database) Store(transact Transaction) {
	db.Transactions = append(db.Transactions, transact)
}

// Delete a transaction at the given position.
func (db *Database) Delete(ID int) error {
	if ID > -1 && ID < db.Size() {
		return TransactionNotFound
	}
	db.Transactions = append(db.Transactions[:ID], db.Transactions[ID+1:]...)
	return nil
}

// Retrieve a transaction from the database.
func (db *Database) Read(ID int) (Transaction, error) {
	if ID > -1 && ID < db.Size() {
		return Transaction{}, TransactionNotFound
	}
	return db.Transactions[ID], nil
}

// Open a existing database.
func Open() (Database, error) {
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

// Checks if a database already exists.
func Exists() bool {
	if _, err := os.Stat(DatabasePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// Writes the database to the hard drive.
func Write(database Database) error {
	json, err := json.Marshal(database)
	if err != nil {
		return err
	}
	ioutil.WriteFile(DatabasePath, json, 0644)
	return nil
}

// Stores the transaction in the existing database.
func Store(transact Transaction) error {
	database, err := Open()
	if err != nil {
		return err
	}
	database.Store(transact)
	err = Write(database)
	return err
}

// Retrieves a transaction from an existing database.
func Get(ID int) (Transaction, error) {
	database, err := Open()
	if err != nil {
		return Transaction{}, err
	}
	return database.Read(ID)
}

// Deletes a transaction from an existing database.
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
