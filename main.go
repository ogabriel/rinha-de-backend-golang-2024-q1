package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&pool_min_conns=%s&pool_max_conns=%[6]s",
		getEnv("DATABASE_USER"),
		getEnv("DATABASE_PASS"),
		getEnv("DATABASE_HOST"),
		getEnv("DATABASE_PORT"),
		getEnv("DATABASE_NAME"),
		getEnv("DATABASE_POOL"),
	)

	config, err := pgxpool.ParseConfig(connString)

	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)

	if err != nil {
		panic(err)
	}

	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		panic(err)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	router.POST("/clientes/:id/transacoes", transacoes(pool))
	router.GET("/clientes/:id/extrato", extrato(pool))

	router.Run("127.0.0.1:" + getEnv("PORT"))
}

func getEnv(envName string) string {
	env, ok := os.LookupEnv(envName)

	if !ok {
		message := fmt.Sprintf("env not declared $%s", envName)
		panic(message)
	}

	return env
}

type Cliente struct {
	ID     int `json:"id"`
	Saldo  int `json:"saldo"`
	Limite int `json:"limite"`
}

type Transacao struct {
	Valor       int    `json:"valor"`
	Tipo        string `json:"tipo"`
	Descricao   string `json:"descricao"`
	RealizadaEm string `json:"realizada_em"`
}

func transacoes(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
	}
}

func validFields(transacao *Transacao) bool {
	if transacao.Valor <= 0 {
		return false
	}

	if transacao.Tipo != "c" && transacao.Tipo != "d" {
		return false
	}

	if tamanhoDescricao := len(transacao.Descricao); tamanhoDescricao > 10 || tamanhoDescricao < 1 {
		return false
	}

	return true
}

func extrato(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
	}
}
