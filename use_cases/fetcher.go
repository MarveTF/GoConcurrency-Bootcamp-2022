package use_cases

import (
	"fmt"
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

func (f Fetcher) Generator(from, to int) (<-chan models.Pokemon, error) {
	// Generate Channel and WaitGroup
	ch := make(chan models.Pokemon)
	wg := sync.WaitGroup{}

	for id := from; id <= to; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pokemon, err := f.api.FetchPokemon(id)
			if err != nil {
				log.Println("Error Fetching Pokemon: ", err)
				return
			}
			fmt.Printf("Pokemon with ID: %v \n: %v", id, pokemon)
			var flatAbilities []string
			for _, t := range pokemon.Abilities {
				flatAbilities = append(flatAbilities, t.Ability.URL)
			}
			pokemon.FlatAbilityURLs = strings.Join(flatAbilities, "|")
			ch <- pokemon
		}(id)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch, nil
}

func (f Fetcher) PokemonGeneratorWriter(pokemons []models.Pokemon) {
	f.storage.Write(pokemons)
}
