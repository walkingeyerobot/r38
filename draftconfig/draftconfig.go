package draftconfig

// DraftConfig stores is directly imported from the set json file.
type DraftConfig struct {
	Hoppers []HopperDefinition `json:"hoppers"`
	Flags   []string           `json:"flags"`
	Cards   []Card             `json:"cards"`
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

func GetCards(cfg DraftConfig) []Card {
	return cfg.Cards
}
