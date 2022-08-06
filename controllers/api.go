package controllers

import (
	"context"
	"fmt"
	"net/http"

	"GoConcurrency-Bootcamp-2022/models"

	"github.com/gin-gonic/gin"
)

type API struct {
	fetcher
	refresher
	getter
}

func NewAPI(fetcher fetcher, refresher refresher, getter getter) API {
	return API{fetcher, refresher, getter}
}

type fetcher interface {
	Generator(done <-chan interface{}, from, to int) <-chan models.Response
}

type refresher interface {
	FanOutFanInCaller(ctx context.Context) error
}

type getter interface {
	GetPokemons(context.Context) ([]models.Pokemon, error)
}

//FillCSV fill the local CSV with data from PokeAPI. By default will fetch from id 1 to 10 unless there are other information on the body
func (api API) FillCSV(c *gin.Context) {
	requestBody := struct {
		From int `json:"from"`
		To   int `json:"to"`
	}{1, 10}
	if err := c.Bind(&requestBody); err != nil {
		c.Status(http.StatusBadRequest)
		fmt.Println(err)
		return
	}
	pokemons := []models.Pokemon{}
	done := make(chan interface{})
	resp := api.Generator(done, requestBody.From, requestBody.To)
	for r := range resp {
		if r.Error != nil {
			fmt.Println("Error on Fill CSV: ", r.Error)
			c.Status(http.StatusInternalServerError)
			return
		}
		pokemons = append(pokemons, r.Pokemon)
	}
	c.Status(http.StatusOK)
}

//RefreshCache feeds the csv data and save in redis
func (api API) RefreshCache(c *gin.Context) {
	err := api.refresher.FanOutFanInCaller(c)
	if err != nil {
		fmt.Println("Error on refresh Cache::", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

//GetPokemons return all pokemons in cache
func (api API) GetPokemons(c *gin.Context) {
	pokemons, err := api.getter.GetPokemons(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, pokemons)
}
