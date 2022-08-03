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

type ResponseResult struct {
	Error   error
	Pokemon models.Pokemon
}

func (f Fetcher) Generator(done <-chan interface{}, from, to int) (<-chan models.Pokemon, error) {
	// Generate Channel and WaitGroup
	resultCh := make(chan models.Pokemon)
	wg := sync.WaitGroup{}

	for id := from; id <= to; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pokemon, err := f.api.FetchPokemon(id)
			resp := ResponseResult{Error: err, Pokemon: pokemon}
			if resp.Error != nil {
				log.Println("Error Fetching Pokemon: ", resp.Error)
				return
			}
			var flatAbilities []string
			for _, t := range resp.Pokemon.Abilities {
				flatAbilities = append(flatAbilities, t.Ability.URL)
			}
			resp.Pokemon.FlatAbilityURLs = strings.Join(flatAbilities, "|")
			select {
			case <-done:
				return
			case resultCh <- resp.Pokemon:
			}
		}(id)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	return resultCh, nil
}

func (f Fetcher) PokemonGeneratorWriter(pokemons []models.Pokemon) {
	f.storage.Write(pokemons)
}
