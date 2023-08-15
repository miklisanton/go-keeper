package main

import (
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Transaction struct{
    ID int`json:"id"`
    Username string`json:"username"`
    Name string`json:"name"`
    Value int`json:"value"`
    Currency string`json:"currency"`
    Created_at time.Time`json:"created_at"`
    Category string`json:"category"`
}

type User struct{
    Name                string`json:"username"`
    PasswordEncrypted   string`json:"-"`
}

type CreateTransactionRequest struct{
    Name        string`json:"name"`
    Username        string`json:"username"`
    Value       int`json:"value"`
    Currency    string`json:"currency"`
}

type CreateUserRequest struct{
    Name        string`json:"username"`
    Password    string`json:"password"`
}

type CreateUserResponse struct{
    Name        string`json:"username"`
    Token    string`json:"token"`
}

func NewTransaction(name string, value int, currency string, category string, username string) *Transaction {
    return &Transaction{
        ID: rand.Intn(10000), 
        Username: username,
        Name: name,
        Currency: currency,
        Value: value,
        Created_at: time.Now().UTC(),
        Category: category,
    }
}

func NewUser(name string, password string) (*User, error) {
    encpw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }
    if name == "" {
        return nil, fmt.Errorf("Name can't be empty!")
    }
    if password == "" {
        return nil, fmt.Errorf("Password can't be empty!")
    }
    return &User{
        Name:               name,
        PasswordEncrypted:  string(encpw),
    }, nil
}

func (u *User)validPassword(password string) bool {
    return bcrypt.CompareHashAndPassword([]byte(u.PasswordEncrypted), []byte(password)) == nil
}
