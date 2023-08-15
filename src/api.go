package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/julienschmidt/httprouter"
)

type APIServer struct{
    Port string
    Storage Storage 
}

type APIError struct {
    Error string`json:"error"`
}

type APIFunc func(http.ResponseWriter, *http.Request, httprouter.Params) error

func newServer(port string, storage Storage) *APIServer {
    return &APIServer{Port: ":"+port, Storage: storage}
}

func (s *APIServer) Run() {
    r := httprouter.New()
    
    log.Println("Server is running on port: ", s.Port)
    r.GET("/transaction", withJWTAuth(makeHTTPHandler(s.handleGetTransaction), s.Storage))
    r.GET("/transaction/:category", withJWTAuth(makeHTTPHandler(s.handleGetTransactionCategory), s.Storage))
    r.GET("/transaction/:category/:id", withJWTAuth(makeHTTPHandler(s.handleGetTransactionID), s.Storage))
    r.POST("/transaction/:category", makeHTTPHandler(s.handlePostTransaction))
    r.DELETE("/transaction/:category", makeHTTPHandler(s.handleDeleteTransaction))

    r.POST("/user", makeHTTPHandler(s.handleCreateUser))
    log.Fatal(http.ListenAndServe(s.Port, r))
}

func makeHTTPHandler(f APIFunc) httprouter.Handle {
   return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
        if err := f(w, r, ps); err != nil {
            WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
        }
    }
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
    w.Header().Add("Content-type", "application/json")
    w.WriteHeader(status)
    return json.NewEncoder(w).Encode(v)
}

func (s *APIServer) handleGetTransactionID(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
    id, err := strconv.Atoi(ps.ByName("id"))
    if err != nil {
        return err
    }
    transaction, err := s.Storage.GetTransactionID(id)
    if err != nil {
        return err
    }
    return WriteJSON(w, http.StatusOK, transaction)
}

func (s *APIServer) handleGetTransaction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
    username, err := getUsername(r)
    if err != nil {
        return err
    }
    transactions, err := s.Storage.GetTransactions(username)
    if err != nil {
        return err
    }
    return WriteJSON(w, http.StatusOK, transactions)
}

func (s *APIServer) handleGetTransactionCategory(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
    username, err := getUsername(r)
    if err != nil {
        return err
    }

    category := ps.ByName("category")
    transactions, err := s.Storage.GetTransactionsByCategory(username, category)
    if err != nil {
        return err
    }
    return WriteJSON(w, http.StatusOK, transactions)
}

func (s *APIServer) handlePostTransaction(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
    CreateTransactionReq := CreateTransactionRequest{}
    if err := json.NewDecoder(r.Body).Decode(&CreateTransactionReq); err != nil{
        return err
    }
    defer r.Body.Close()
    
    transaction := NewTransaction(CreateTransactionReq.Name,
                                CreateTransactionReq.Value,
                                CreateTransactionReq.Currency,
                                ps.ByName("category"),
                                CreateTransactionReq.Username)
    id, err := s.Storage.CreateTransaction(transaction) 
    if err != nil {
        return err
    }
    transaction.ID = id;
    return WriteJSON(w, http.StatusOK, transaction)
}

func (s *APIServer) handleDeleteTransaction(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
    id, err := strconv.Atoi(ps.ByName("id"))
    if err != nil {
        return err
    }
    return s.Storage.DeleteTransaction(id)
}

func (s *APIServer) handleCreateUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
    CreateUserReq := CreateUserRequest{}
    if err := json.NewDecoder(r.Body).Decode(&CreateUserReq); err != nil {
        return err
    }
    user, err := NewUser(CreateUserReq.Name, CreateUserReq.Password)
    if err != nil {
        return err
    }
    if err := s.Storage.CreateUser(user); err != nil {
        return err
    }
    token, err := generateJWT(user)
    if err != nil {
        return err
    }
    userResponse := CreateUserResponse{Name: user.Name,
                                        Token: token}
    cookie := &http.Cookie{
        Name:       "jwtToken",
        Value:      token,
        Expires: time.Now().Add(24 *time.Hour),
    }
    http.SetCookie(w, cookie)
    return WriteJSON(w, http.StatusOK, userResponse)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
    userReq := CreateUserRequest{}
    if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
        return err
    }
    defer r.Body.Close()

    user, err := s.Storage.GetUser(userReq.Name)
    if err != nil {
        return err
    }
    if !user.validPassword(user.PasswordEncrypted) {
        return fmt.Errorf("not authenticated")
    }
        
    token, err := generateJWT(user)
    if err != nil {
        return err
    }
    userResponse := CreateUserResponse{Name: user.Name,
                                        Token: token}
    cookie := &http.Cookie{
        Name:       "jwtToken",
        Value:      token,
        Expires: time.Now().Add(24 *time.Hour),
    }
    http.SetCookie(w, cookie)
    return WriteJSON(w, http.StatusOK, userResponse)
}

func withJWTAuth(handlerFunc httprouter.Handle, storage Storage) httprouter.Handle {
    return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
        cookie, err := r.Cookie("jwtToken")
        if err != nil {
            permissionDenied(w)
            return
        }

        tokenStr := cookie.Value
        token, err := validateJWT(tokenStr)
        if err != nil {
            permissionDenied(w)
            return
        }
        if !token.Valid {
            permissionDenied(w)
            return
        }

        username, err := getUsername(r)
        if err != nil {
            permissionDenied(w)
            return
        }
        claims := token.Claims.(jwt.MapClaims)
        if username != claims["username"] {
            permissionDenied(w)
            return
        }
        handlerFunc(w, r, ps)
    }
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
}

func generateJWT(user *User) (string, error) {
    claims := jwt.MapClaims{
        "expiresAt":    15000,
        "username":     user.Name,
    }
    secret := os.Getenv("JWT_SECRET")
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

    return token.SignedString([]byte(secret))
}

func permissionDenied(w http.ResponseWriter) {
    WriteJSON(w, http.StatusForbidden, APIError{Error: "permission denied"})
}

func getUsername(r *http.Request) (string, error){
    CreateTransactionReq := CreateTransactionRequest{}
    body, err := io.ReadAll(r.Body)
    if err != nil {
        return "", err
    }
    if err := json.Unmarshal(body, &CreateTransactionReq); err != nil{
        return "", err
    }
    r.Body = io.NopCloser(bytes.NewReader(body))
    return CreateTransactionReq.Username, nil
}



