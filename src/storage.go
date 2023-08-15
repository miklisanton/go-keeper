package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)


type Storage interface {
    GetTransactionID(int) (*Transaction, error)
    GetTransactions(string) ([]*Transaction, error)
    GetTransactionsByCategory(string, string) ([]*Transaction, error)
    //updateTransaction(int)
    CreateTransaction(*Transaction) (int, error)
    DeleteTransaction(int) error
    CreateUser(*User) error
    GetUser(string) (*User, error)
}

type PostgresStore struct {
    db *sql.DB
}

func newPostgresStore() (*PostgresStore, error) {
    connStr := "user=postgres dbname=postgres password=Aa36423642 sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }
    if err := db.Ping();err != nil {
        return nil, err
    }
    return &PostgresStore{
        db: db,
    }, nil
}

func (s *PostgresStore)createTransactionTable() error {
    query := `create table if not exists transaction (
        id serial primary key,
        name varchar(50) not null,
        value int not null,
        currency char(3) not null,
        created_at timestamp default current_timestamp,
        category varchar(50),
        username varchar(50),
        constraint fk_username foreign key(username) references users(name) on delete cascade
    );`
    _, err := s.db.Exec(query);
    return err
}

func (s *PostgresStore)createUserTable() error {
    query := `create table if not exists users (
        name varchar(50) primary key not null,
        passwordenc varchar(100)
    );`
    _, err := s.db.Exec(query);
    return err
}

func (s *PostgresStore)Init() error {
    err := s.createUserTable()
    if err != nil {
        return err
    }
    return s.createTransactionTable()
}


func (s *PostgresStore)CreateTransaction(tr *Transaction) (int, error) {
    query := `insert into transaction
    (name, value, currency, created_at, category, username)
    values ($1, $2, $3, $4, $5, $6) returning id`
    var newID int
    err := s.db.QueryRow(
        query,
        tr.Name,
        tr.Value,
        tr.Currency,
        tr.Created_at,
        tr.Category,
        tr.Username,
    ).Scan(&newID)

    return newID, err
}

func (s *PostgresStore) GetTransactionID(id int) (*Transaction, error) {
    query := `select id, name, value, currency, created_at, category from transaction
    where id = $1`

    rows, err := s.db.Query(query, id)
    if err != nil {
        return nil, err
    }
    for rows.Next() {
        return scanIntoTransaction(rows) 
    }
    return nil, fmt.Errorf("Transaction %d not found", id)
}

func (s *PostgresStore) GetTransactions(username string) ([]*Transaction, error) {
    query := `select id, name, value, currency, created_at, category, username from transaction
                where username = $1`

    rows, err := s.db.Query(query, username)
    if err != nil {
        return nil, err
    }

    transactions := []*Transaction{}
    for rows.Next() {
        transaction, err := scanIntoTransaction(rows)
        if err != nil {
            return nil, err
        }
        transactions = append(transactions, transaction)
    }
    return transactions, nil
}

func (s *PostgresStore) GetTransactionsByCategory(username string, category string) ([]*Transaction, error) {
    query := `select id, name, value, currency, created_at, category, username from transaction
                where username = $1 and category = $2`

    rows, err := s.db.Query(query, username, category)
    if err != nil {
        return nil, err
    }

    transactions := []*Transaction{}
    for rows.Next() {
        transaction, err := scanIntoTransaction(rows)
        if err != nil {
            return nil, err
        }
        transactions = append(transactions, transaction)
    }
    return transactions, nil
}


func (s *PostgresStore) DeleteTransaction(id int) error {
    _, err := s.db.Query("delete from transaction where id = $1", id)
    return err
}

func (s *PostgresStore) CreateUser(user *User) error {
    query := `insert into users (name, passwordenc) values ($1, $2)`
    _, err := s.db.Query(query, user.Name, user.PasswordEncrypted) 
    if err != nil{
        if strings.Contains(err.Error(), "unique constraint") {
            return fmt.Errorf("Username %s already exists", user.Name)
        }
    }
    return err
}

func (s *PostgresStore) GetUser(name string) (*User, error) {
    query := `select name, passwordenc from users where name = $1`  

    rows, err := s.db.Query(query, name)
    if err != nil {
        return nil, err
    }
    for rows.Next() {
        return scanIntoUser(rows) 
    }
    return nil, fmt.Errorf("User does not exist %s", name)
}

func scanIntoTransaction(rows *sql.Rows) (*Transaction, error) {
    transaction := Transaction{}
    err := rows.Scan(
        &transaction.ID,
        &transaction.Name,
        &transaction.Value,
        &transaction.Currency,
        &transaction.Created_at,
        &transaction.Category,
        &transaction.Username,
    )
    return &transaction, err
}

func scanIntoUser(rows *sql.Rows) (*User, error) {
    user := User{}
    err := rows.Scan(
        &user.Name,
        &user.PasswordEncrypted,
    )
    return &user, err
}
