package use_cases

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"GoConcurrency-Bootcamp-2022/models"
)

type reader interface {
	Read() ([]models.Pokemon, error)
}

type saver interface {
	Save(context.Context, []models.Pokemon) error
}

type fetcher interface {
	FetchAbility(string) (models.Ability, error)
}

type Refresher struct {
	reader
	saver
	fetcher
}

func NewRefresher(reader reader, saver saver, fetcher fetcher) Refresher {
	return Refresher{reader, saver, fetcher}
}

func (r Refresher) FanInCaller(ctx context.Context) error {
	pokemons, err := r.Read()
	if err != nil {
		return err
	}
	inputCh := r.GenerateWork(pokemons)
	queuedCh := r.FanIn(inputCh)

	resp1 := r.FanOut(queuedCh)
	resp2 := r.FanOut(queuedCh)
	resp3 := r.FanOut(queuedCh)

	toSavePokemons := []models.Pokemon{}
	for range pokemons {

		select {
		case resp := <-resp1:
			if resp.Error != nil {
				fmt.Println("Error on fan in caller::", resp.Error)
				return resp.Error
			}
			fmt.Println("Worker1:\n", resp.Pokemon)
			toSavePokemons = append(toSavePokemons, resp.Pokemon)

		case resp := <-resp2:
			if resp.Error != nil {
				fmt.Println("Error on fan in caller::", resp.Error)
				return resp.Error
			}
			fmt.Println("Worker2:\n", resp.Pokemon)
			toSavePokemons = append(toSavePokemons, resp.Pokemon)

		case resp := <-resp3:
			if resp.Error != nil {
				fmt.Println("Error on fan in caller::", resp.Error)
				return resp.Error
			}
			fmt.Println("Worker3:\n", resp.Pokemon)
			toSavePokemons = append(toSavePokemons, resp.Pokemon)
		}

		err := r.Save(ctx, toSavePokemons)
		if err != nil {
			fmt.Println("error on ABILITY GATHERING::", err)
			return err
		}
	}
	return nil
}

func (fir Refresher) GenerateWork(pokemons []models.Pokemon) <-chan models.Pokemon {
	ch := make(chan models.Pokemon)

	go func() {
		defer close(ch)

		for _, p := range pokemons {
			ch <- p
		}
	}()

	return ch
}

func (r Refresher) FanIn(inputs ...<-chan models.Pokemon) <-chan models.Pokemon {
	var wg sync.WaitGroup
	outputCh := make(chan models.Pokemon)
	wg.Add(len(inputs))

	for _, input := range inputs {
		// opening a go routine for each channel
		go func(ch <-chan models.Pokemon) {
			for {
				pokemon, ok := <-ch
				if !ok {
					wg.Done()
					break
				}

				outputCh <- pokemon
			}
		}(input)
	}

	//closing the channel
	go func() {
		wg.Wait()
		close(outputCh)
	}()

	return outputCh
}

func (r Refresher) FanOut(in <-chan models.Pokemon) <-chan models.Response {

	outputCh := make(chan models.Response)
	resp := models.Response{}
	go func(ch <-chan models.Pokemon) {
		for pokemon := range in {
			//obtaining the abilities
			urls := strings.Split(pokemon.FlatAbilityURLs, "|")
			var abilities []string
			for _, url := range urls {
				ability, err := r.FetchAbility(url)
				resp := models.Response{Error: err}
				if resp.Error != nil {
					fmt.Println("error on ABILITY GATHERING::", resp.Error)
				}
				for _, ee := range ability.EffectEntries {
					abilities = append(abilities, ee.Effect)
				}
			}

			pokemon.EffectEntries = abilities
			resp = models.Response{Pokemon: pokemon}
			outputCh <- resp
		}
	}(in)

	return outputCh
}
