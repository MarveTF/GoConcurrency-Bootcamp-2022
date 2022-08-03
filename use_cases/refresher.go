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

	out1 := r.FanOut(queuedCh)
	out2 := r.FanOut(queuedCh)
	out3 := r.FanOut(queuedCh)

	toSavePokemons := []models.Pokemon{}
	for range pokemons {

		select {
		case value := <-out1:
			fmt.Println("Worker1:\n", value)
			toSavePokemons = append(toSavePokemons, value)

		case value := <-out2:
			fmt.Println("Worker2:\n", value)
			toSavePokemons = append(toSavePokemons, value)

		case value := <-out3:
			fmt.Println("Worker3:\n", value)
			toSavePokemons = append(toSavePokemons, value)
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

func (r Refresher) FanIn(inputs ...<-chan models.Pokemon) <-chan models.Pokemon { // confirm channels
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

func (r Refresher) FanOut(in <-chan models.Pokemon) <-chan models.Pokemon { // confirm channels

	outputCh := make(chan models.Pokemon)

	go func(ch <-chan models.Pokemon) {
		for pokemon := range in {
			//obtaining the abilities
			urls := strings.Split(pokemon.FlatAbilityURLs, "|")
			var abilities []string
			for _, url := range urls {
				ability, err := r.FetchAbility(url)
				resp := models.Response{Error: err, Pokemon: pokemon}
				if resp.Error != nil {
					fmt.Println("error on ABILITY GATHERING::", resp.Error)
					return
				}
				for _, ee := range ability.EffectEntries {
					abilities = append(abilities, ee.Effect)
				}
			}

			pokemon.EffectEntries = abilities

			outputCh <- pokemon
		}
	}(in)

	return outputCh
}
