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

func (r Refresher) FanOutFanInCaller(ctx context.Context) error {
	// Reading de pokemons from CSV file.
	pokemons, err := r.Read()
	if err != nil {
		return err
	}
	inputCh := r.GenerateWork(pokemons)

	resp1 := r.FanOut(inputCh)
	resp2 := r.FanOut(inputCh)
	resp3 := r.FanOut(inputCh)

	respCh := r.FanIn(resp1, resp2, resp3)
	toSavePokemons := []models.Pokemon{}

	for resp := range respCh {
		if resp.Error != nil {
			fmt.Println("Error on fan in caller::", resp.Error)
			return resp.Error
		}
		toSavePokemons = append(toSavePokemons, resp.Pokemon)
	}
	err = r.Save(ctx, toSavePokemons)
	if err != nil {
		fmt.Println("error on Saving Abilities::", err)
		return err

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

func (r Refresher) FanIn(inputs ...<-chan models.Response) <-chan models.Response {
	var wg sync.WaitGroup
	outputCh := make(chan models.Response)
	wg.Add(len(inputs))

	for _, input := range inputs {
		// opening a go routine for each channel
		go func(ch <-chan models.Response) {
			for {
				response, ok := <-ch
				if !ok {
					wg.Done()
					break
				}
				outputCh <- response
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
		defer close(outputCh)
		for pokemon := range in {
			//obtaining the abilities
			urls := strings.Split(pokemon.FlatAbilityURLs, "|")
			var abilities []string
			for _, url := range urls {
				ability, err := r.FetchAbility(url)
				resp := models.Response{Error: err}
				if resp.Error != nil {
					fmt.Println("error on Ability Gathering::", resp.Error)
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
