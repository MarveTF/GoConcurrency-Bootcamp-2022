package use_cases

import (
	"log"
	"strings"
	"sync"

	"GoConcurrency-Bootcamp-2022/models"
)

type api interface {
	FetchPokemon(id int) (models.Pokemon, error)
}

type writer interface {
	Write(pokemons []models.Pokemon) error
}

type Fetcher struct {
	api     api
	storage writer
}

func NewFetcher(api api, storage writer) Fetcher {
	return Fetcher{api, storage}
}

var pokemons []models.Pokemon

func (f Fetcher) Generator(done <-chan interface{}, from, to int) <-chan models.Response {
	// Generate Channel and WaitGroup
	resultCh := make(chan models.Response)
	wg := sync.WaitGroup{}

	for id := from; id <= to; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pokemon, err := f.api.FetchPokemon(id)
			resp := models.Response{Error: err, Pokemon: pokemon}
			if resp.Error != nil {
				log.Println("Error Fetching Pokemon on Generator: ", resp.Error)
			}
			var flatAbilities []string
			for _, t := range resp.Pokemon.Abilities {
				flatAbilities = append(flatAbilities, t.Ability.URL)
			}
			resp.Pokemon.FlatAbilityURLs = strings.Join(flatAbilities, "|")
			pokemons = append(pokemons, resp.Pokemon)
			select {
			case <-done:
				return
			case resultCh <- resp:
			}
		}(id)
	}

	go func() {
		wg.Wait()
		close(resultCh)
		// Waiting for the channel to be closed to write the saved pokemons.
		f.storage.Write(pokemons)
	}()

	return resultCh
}
