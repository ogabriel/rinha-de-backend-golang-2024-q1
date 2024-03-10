package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&pool_min_conns=%s&pool_max_conns=%s",
		getEnv("DATABASE_USER"),
		getEnv("DATABASE_PASS"),
		getEnv("DATABASE_HOST"),
		getEnv("DATABASE_PORT"),
		getEnv("DATABASE_NAME"),
		getEnv("DATABASE_POOL"),
		getEnv("DATABASE_POOL"),
	)

	config, err := pgxpool.ParseConfig(connString)
	config.HealthCheckPeriod = 10 * time.Minute

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

type Transacao struct {
	Valor       int    `json:"valor"`
	Tipo        string `json:"tipo"`
	Descricao   string `json:"descricao"`
	RealizadaEm string `json:"realizada_em"`
}

type TransacoesResponse struct {
	Saldo  int `json:"saldo"`
	Limite int `json:"limite"`
}

func transacoes(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id, err := strconv.Atoi(ctx.Param("id"))

		if id < 1 || err != nil {
			ctx.Status(http.StatusNotFound)
			return
		}

		var transacao Transacao

		if err := ctx.ShouldBindJSON(&transacao); err != nil {
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		if !validFields(&transacao) {
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		var credito_ou_debito int

		if transacao.Tipo == "c" {
			credito_ou_debito = transacao.Valor
		} else {
			credito_ou_debito = -transacao.Valor
		}

		tx, err := pool.Begin(ctx)

		if err != nil {
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		var saldo, limite int

		if err := tx.QueryRow(ctx, "SELECT saldo, limite FROM clientes WHERE id = $1 FOR UPDATE", id).Scan(&saldo, &limite); err != nil {
			tx.Rollback(ctx)
			ctx.Status(http.StatusNotFound)
			return
		}

		novo_saldo := saldo + credito_ou_debito

		if novo_saldo < -limite {
			tx.Rollback(ctx)
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		realizada_em := time.Now().Format("2006-01-02T15:04:05.999999Z")

		if _, err := tx.Exec(ctx, "INSERT INTO transacoes (valor, tipo, descricao, realizada_em, cliente_id) VALUES ($1, $2, $3, $4, $5)", transacao.Valor, transacao.Tipo, transacao.Descricao, realizada_em, id); err != nil {
			tx.Rollback(ctx)
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		if _, err := tx.Exec(ctx, "UPDATE clientes SET saldo = $1 WHERE id = $2", novo_saldo, id); err != nil {
			tx.Rollback(ctx)
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		transacaoResponse := TransacoesResponse{
			Saldo:  novo_saldo,
			Limite: limite,
		}

		ctx.JSON(http.StatusOK, transacaoResponse)
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

type ExtratoResponse struct {
	Saldo             SaldoResponse             `json:"saldo"`
	UltimasTransacoes []UltimaTransacaoResponse `json:"ultimas_transacoes"`
}

type SaldoResponse struct {
	Total       int    `json:"total"`
	Limite      int    `json:"limite"`
	DataExtrato string `json:"data_extrato"`
}

type UltimaTransacaoResponse struct {
	Valor       int    `json:"valor"`
	Tipo        string `json:"tipo"`
	Descricao   string `json:"descricao"`
	RealizadaEm string `json:"realizada_em"`
}

func extrato(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id, err := strconv.Atoi(ctx.Param("id"))

		if err != nil {
			ctx.Status(http.StatusNotFound)
			return
		}

		saldoResponse := SaldoResponse{}

		conn, err := pool.Acquire(ctx)

		if err != nil {
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		defer conn.Release()

		if err := conn.QueryRow(ctx, "SELECT saldo, limite FROM clientes WHERE id = $1", id).Scan(&saldoResponse.Total, &saldoResponse.Limite); err != nil {
			conn.Release()
			ctx.Status(http.StatusNotFound)
			return
		}

		rows, err := conn.Query(ctx, "SELECT valor, tipo, descricao, realizada_em FROM transacoes WHERE cliente_id = $1 ORDER BY id DESC LIMIT 10", id)

		conn.Release()

		if err != nil {
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		ultimasTransacoesResponse, err := pgx.CollectRows(rows, pgx.RowToStructByPos[UltimaTransacaoResponse])

		if err != nil {
			conn.Release()
			ctx.Status(http.StatusUnprocessableEntity)
			return
		}

		saldoResponse.DataExtrato = time.Now().Format("2006-01-02T15:04:05.999999Z")

		extratoResponse := ExtratoResponse{
			Saldo:             saldoResponse,
			UltimasTransacoes: ultimasTransacoesResponse,
		}

		ctx.JSON(http.StatusOK, extratoResponse)
	}
}
