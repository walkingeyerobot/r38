package main

import (
	"fmt"
	"math/rand"
)

// DO NOT add a non-simple type to this struct.
// if you do, copying cards with a simple assignment will break (I think).
type Card struct {
	Mtgo          string
	Number        string
	Rarity        string
	Name          string
	Color         string
	ColorIdentity string
	Cmc           int64
	Type          string
	Rating        float64
	Foil          bool
}

type CardSet struct {
	Mythics   []Card
	Rares     []Card
	Uncommons []Card
	Commons   []Card
	Basics    []Card
}

type Hopper interface {
	Refill()
	Pop() (Card, bool)
}

type NormalHopper struct {
	Cards  []Card
	Source []Card
}

type FoilHopper struct {
	OtherHopper *Hopper
	Cards       []Card
	Source      []Card
}

type BasicLandHopper struct {
	Cards  []Card
	Source []Card
}

func (h *NormalHopper) Pop() (Card, bool) {
	ret := h.Cards[0]
	h.Cards = h.Cards[1:]
	return ret, len(h.Cards) == 0
}

func (h *FoilHopper) Pop() (Card, bool) {
	var ret Card
	var empty bool

	if rand.Intn(4) == 0 {
		ret = h.Cards[0]
		h.Cards = h.Cards[1:]
		empty = len(h.Cards) == 0
	} else {
		ret, empty = (*h.OtherHopper).Pop()
	}

	return ret, empty
}

func (h *BasicLandHopper) Pop() (Card, bool) {
	ret := h.Cards[0]
	h.Cards = h.Cards[1:]
	if len(h.Cards) == 0 {
		h.Refill()
	}
	return ret, false
}

func (h *NormalHopper) Refill() {
	for _, v := range h.Source {
		var copiedCard Card
		copiedCard = v // this copies???
		h.Cards = append(h.Cards, copiedCard)
	}
	rand.Shuffle(len(h.Cards), func(i, j int) {
		h.Cards[i], h.Cards[j] = h.Cards[j], h.Cards[i]
	})
}

func (h *FoilHopper) Refill() {
	for _, v := range h.Source {
		var copiedCard Card
		copiedCard = v // this copies???
		h.Cards = append(h.Cards, copiedCard)
	}
	rand.Shuffle(len(h.Cards), func(i, j int) {
		h.Cards[i], h.Cards[j] = h.Cards[j], h.Cards[i]
	})
}

func (h *BasicLandHopper) Refill() {
	for _, v := range h.Source {
		var copiedCard Card
		copiedCard = v // this copies???
		h.Cards = append(h.Cards, copiedCard)
	}
	// no need to shuffle
}

func MakeNormalHopper(sources ...[]Card) *NormalHopper {
	ret := NormalHopper{}
	for _, cardList := range sources {
		for _, v := range cardList {
			var copiedCard Card
			copiedCard = v // this copies???
			ret.Source = append(ret.Source, copiedCard)
		}
	}
	ret.Refill()
	return &ret
}

func MakeUncommonHopper(sources ...[]Card) *NormalHopper {
	ret := NormalHopper{}
	for _, cardList := range sources {
		for _, v := range cardList {
			var copiedCard Card
			copiedCard = v // this copies???
			ret.Source = append(ret.Source, copiedCard)
		}
	}
	ret.Refill()
	Shuffle(ret.Cards)
	return &ret
}

func MakeFoilHopper(other *Hopper, sources ...[]Card) *FoilHopper {
	ret := FoilHopper{OtherHopper: other}
	for _, cardList := range sources {
		for _, v := range cardList {
			var copiedCard Card
			copiedCard = v // this copies???
			copiedCard.Foil = true
			ret.Source = append(ret.Source, copiedCard)
		}
	}
	ret.Refill()
	return &ret
}

func MakeBasicLandHopper(sources ...[]Card) *BasicLandHopper {
	ret := BasicLandHopper{}
	for _, cardList := range sources {
		for _, v := range cardList {
			var copiedCard Card
			copiedCard = v // this copies???
			ret.Source = append(ret.Source, copiedCard)
		}
	}
	ret.Refill()
	return &ret
}

func Shuffle(cards []Card) {
	minDistance := 5
	fmt.Printf("~~~~~~~~~~STARTING NEW SHUFFLE~~~~~~~~~~\n")
	cardCount := len(cards)
	passes := true
	outerTries := 0
	for {
		passes = true
		outerTries++
		fmt.Printf("=====BEGINING ATTEMPT %d=====\n", outerTries)
		for i := cardCount - 1; i >= 0; i-- {
			fmt.Printf("------%d,%d\n", i, cardCount)
			var j int
			innerTries := 0
			if i == 0 {
				fmt.Printf("^^^^^^\n")
				j = 0
				fmt.Printf("%d<=>%d\t%s\n", i, j, cards[j].Name)
				for k := i + 1; k <= i+minDistance; k++ {
					fmt.Printf("\t%d", k)
					if cards[j].Name == cards[k].Name {
						fmt.Printf("\tfails!")
						passes = false
					} else {
						fmt.Printf("\tpasses")
					}
					fmt.Printf("\t%s\n", cards[k].Name)
					if !passes {
						break
					}
				}
			} else if i == 1 {
				fmt.Printf("&&&&&&\n")
				j = rand.Intn(i)
				fmt.Printf("%d<=>%d\t%s\n", i, j, cards[j].Name)
				for k := i + 1; k <= i+minDistance; k++ {
					fmt.Printf("\t%d", k)
					if cards[j].Name == cards[k].Name {
						fmt.Printf("\tfails!")
						passes = false
					} else {
						fmt.Printf("\tpasses")
					}
					fmt.Printf("\t%s\n", cards[k].Name)
					if !passes {
						break
					}
				}
			} else if i != cardCount-1 {
				fmt.Printf("******\n")
				for {
					passes = true
					innerTries++
					j = rand.Intn(i)
					fmt.Printf("%d<=>%d\t%s\n", i, j, cards[j].Name)
					for k := i + 1; k < cardCount && k <= i+minDistance; k++ {
						fmt.Printf("\t%d", k)
						if cards[j].Name == cards[k].Name {
							fmt.Printf("\tfails!")
							passes = false
							// break
						} else {
							fmt.Printf("\tpasses")
						}
						fmt.Printf("\t%s\n", cards[k].Name)
						if !passes {
							break
						}
					}
					if passes || innerTries > 100 {
						break
					}
				}
			} else {
				j = rand.Intn(i)
				fmt.Printf("$$$$$$\n")
				fmt.Printf("%d<=>%d\t%s\n", i, j, cards[j].Name)
			}
			if !passes {
				break
			}
			cards[i], cards[j] = cards[j], cards[i]
		}
		if passes || outerTries > 10000 {
			break
		}
	}
	if !passes {
		fmt.Printf("panic after %d tries\n", outerTries)
		panic("cannot shuffle")
	}
	/*
		rand.Shuffle(len(cards), func(i, j int) {
			cards[i], cards[j] = cards[j], cards[i]
		})
	*/
}
