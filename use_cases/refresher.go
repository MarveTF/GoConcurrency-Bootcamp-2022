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
	SaveFanIn(ctx context.Context, pokemon models.Pokemon) error
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
	out := r.FanIn(ctx, inputCh)
	for value := range out {
		fmt.Println("VALUE::: ", value)
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

func (r Refresher) FanIn(ctx context.Context, inputs ...<-chan models.Pokemon) <-chan models.Pokemon { // confirm channels
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
				if err := r.SaveFanIn(ctx, pokemon); err != nil {
					fmt.Println("error on SAVE FAN IN::", err)
					//return err
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

func (r Refresher) Refresh(ctx context.Context) error {
	pokemons, err := r.Read()
	if err != nil {
		return err
	}

	for i, p := range pokemons {
		urls := strings.Split(p.FlatAbilityURLs, "|")
		var abilities []string
		for _, url := range urls {
			ability, err := r.FetchAbility(url)
			if err != nil {
				return err
			}

			for _, ee := range ability.EffectEntries {
				abilities = append(abilities, ee.Effect)
			}
		}

		pokemons[i].EffectEntries = abilities
	}

	if err := r.Save(ctx, pokemons); err != nil {
		return err
	}

	return nil
}
