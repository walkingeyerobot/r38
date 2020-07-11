package main

import (
	"math/rand"
)

// DraftConfig stores is directly imported from the set json file.
type DraftConfig struct {
	Hoppers []HopperDefinition `json:"hoppers"`
	Flags   []string           `json:"flags"`
	Cards   []Card             `json:"cards"`
}

// HopperDefinition is part of DraftConfig and describes hoppers.
type HopperDefinition struct {
	Type string  `json:"type"`
	Refs []int64 `json:"refs"`
}

// Card is part of DraftConfig and describes cards.
// DO NOT add a non-simple type to this struct.
// if you do, copying cards with a simple assignment will break (I think).
type Card struct {
	Color         string  `json:"color"`
	ColorIdentity string  `json:"color_identity"`
	Dfc           bool    `json:"dfc"`
	ID            string  `json:"id"`
	Rarity        string  `json:"rarity"`
	Rating        float64 `json:"rating"`
	Data          string  `json:"data"`
	Foil          bool
}

// CardSet helps us lookup cards by rarity.
type CardSet struct {
	All       []Card
	Mythics   []Card
	Rares     []Card
	Uncommons []Card
	Commons   []Card
	Basics    []Card
}

// Hopper is effectively a stack of cards waiting to be put into packs.
type Hopper interface {
	Refill()
	Pop() (Card, bool)
}

// NormalHopper is a hopper with no special logic.
type NormalHopper struct {
	Cards      []Card
	Source     []Card
	Refillable bool
}

// FoilHopper has a 1/4 chance to return a foil card from its own cards and 3/4 chance to return a non-foil card from OtherHoppers[].
type FoilHopper struct {
	OtherHoppers []*Hopper
	Cards        []Card
	Source       []Card
}

// BasicLandHopper is never empty and always returns a random basic.
type BasicLandHopper struct {
	Cards  []Card
	Source []Card
}

// Pop returns a card from the hopper and reports if the hopper is now empty.
func (h *NormalHopper) Pop() (Card, bool) {
	ret := h.Cards[0]
	h.Cards = h.Cards[1:]
	if h.Refillable && len(h.Cards) == 0 {
		h.Refill()
	}
	return ret, len(h.Cards) == 0
}

// Pop returns a card from the hopper and reports if the hopper is now empty.
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

// Pop returns a card from the hopper and reports if the hopper is now empty.
func (h *BasicLandHopper) Pop() (Card, bool) {
	ret := h.Cards[0]
	h.Cards = h.Cards[1:]
	if len(h.Cards) == 0 {
		h.Refill()
	}
	return ret, false
}

// Refill refills the hopper from its source cards.
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

// Refill refills the hopper from its source cards.
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

// Refill refills the hopper from its source cards.
func (h *BasicLandHopper) Refill() {
	for _, v := range h.Source {
		var copiedCard Card
		copiedCard = v // this copies???
		h.Cards = append(h.Cards, copiedCard)
	}
	// no need to shuffle
}

// MakeNormalHopper creates a NormalHopper.
func MakeNormalHopper(refillable bool, sources ...[]Card) *NormalHopper {
	ret := NormalHopper{}
	for _, cardList := range sources {
		for _, v := range cardList {
			var copiedCard Card
			copiedCard = v // this copies???
			ret.Source = append(ret.Source, copiedCard)
		}
	}
	ret.Refill()
	ret.Refillable = refillable
	return &ret
}

// MakeFoilHopper creates a FoilHopper.
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

// MakeBasicLandHopper creates a BasicLandHopper.
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
