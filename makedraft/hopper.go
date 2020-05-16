package main

import (
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
	OtherHoppers []*Hopper
	Cards        []Card
	Source       []Card
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

	r := rand.Intn(4)
	if r == 3 {
		ret = h.Cards[0]
		h.Cards = h.Cards[1:]
		empty = len(h.Cards) == 0
	} else {
		ret, empty = (*h.OtherHoppers[r]).Pop()
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

func MakeFoilHopper(commonHopper1 *Hopper, commonHopper2 *Hopper, commonHopper3 *Hopper, sources ...[]Card) *FoilHopper {
	ret := FoilHopper{OtherHoppers: []*Hopper{commonHopper1, commonHopper2, commonHopper3}}
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
