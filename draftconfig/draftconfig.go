package draftconfig

import (
	"encoding/json"
	"net/http"
	"strings"
)

// DraftConfig stores is directly imported from the set json file.
type DraftConfig struct {
	Hoppers     []HopperDefinition `json:"hoppers"`
	Flags       []string           `json:"flags"`
	Cards       []Card             `json:"cards"`
	CubeCobraId string             `json:"cube_cobra_id"`
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

type CardData struct {
	Foil      bool             `json:"foil"`
	Scryfall  CardScryfallData `json:"scryfall"`
	ImageUris []string         `json:"image_uris"`
	MtgoId    int              `json:"mtgo_id"`
}

type CardScryfallData struct {
	Cmc             int      `json:"cmc"`
	CollectorNumber string   `json:"collector_number"`
	ColorIdentity   []string `json:"color_identity"`
	Colors          []string `json:"colors"`
	Layout          string   `json:"layout"`
	ManaCost        string   `json:"mana_cost"`
	Name            string   `json:"name"`
	Rarity          string   `json:"rarity"`
	Set             string   `json:"set"`
	TypeLine        string   `json:"type_line"`
}

type CubeCobraCard struct {
	CardId  string               `json:"cardID"`
	Finish  string               `json:"finish"`
	Details CubeCobraCardDetails `json:"details"`
}

type CubeCobraCardDetails struct {
	Cmc             int      `json:"cmc"`
	CollectorNumber string   `json:"collector_number"`
	Colors          []string `json:"colors"`
	ColorIdentity   []string `json:"color_identity"`
	Rarity          string   `json:"rarity"`
	ImageNormal     string   `json:"image_normal"`
	Layout          string   `json:"layout"`
	MtgoId          int      `json:"mtgo_id"`
	Name            string   `json:"name"`
	ParsedCost      []string `json:"parsed_cost"`
	Set             string   `json:"set"`
	Type            string   `json:"type"`
}

type CubeCobraList struct {
	Cards struct {
		Mainboard []CubeCobraCard `json:"mainboard"`
	} `json:"cards"`
}

func ParsedCostToCost(parsedCost []string) string {
	cost := ""
	for _, costComponent := range parsedCost {
		costComponent = strings.ToUpper(costComponent)
		costComponent = strings.ReplaceAll(costComponent, "-", "/")
		cost = cost + "{" + costComponent + "}"
	}
	return cost
}

func GetCards(cfg DraftConfig) ([]Card, error) {
	if len(cfg.Cards) == 0 && len(cfg.CubeCobraId) > 0 {
		resp, err := http.Get("https://cubecobra.com/cube/api/cubeJSON/" + cfg.CubeCobraId)
		if err != nil {
			return nil, err
		}
		var cubeCobraList CubeCobraList
		err = json.NewDecoder(resp.Body).Decode(&cubeCobraList)
		if err != nil {
			return nil, err
		}
		for _, cubeCobraCard := range cubeCobraList.Cards.Mainboard {
			cardData := CardData{
				Foil: cubeCobraCard.Finish == "Foil",
				Scryfall: CardScryfallData{
					Cmc:             cubeCobraCard.Details.Cmc,
					ColorIdentity:   cubeCobraCard.Details.ColorIdentity,
					Layout:          cubeCobraCard.Details.Layout,
					Name:            cubeCobraCard.Details.Name,
					TypeLine:        cubeCobraCard.Details.Type,
					CollectorNumber: cubeCobraCard.Details.CollectorNumber,
					Rarity:          cubeCobraCard.Details.Rarity,
					Set:             cubeCobraCard.Details.Set,
					Colors:          cubeCobraCard.Details.Colors,
					ManaCost:        ParsedCostToCost(cubeCobraCard.Details.ParsedCost),
				},
				ImageUris: []string{cubeCobraCard.Details.ImageNormal},
				MtgoId:    cubeCobraCard.Details.MtgoId,
			}
			cardDataByes, err := json.Marshal(cardData)
			if err != nil {
				return nil, err
			}
			card := Card{
				Color:         strings.Join(cubeCobraCard.Details.Colors, ""),
				ColorIdentity: strings.Join(cubeCobraCard.Details.ColorIdentity, ""),
				Dfc:           cubeCobraCard.Details.Layout == "transform",
				ID:            cubeCobraCard.CardId,
				Rarity:        cubeCobraCard.Details.Rarity,
				Rating:        0,
				Data:          string(cardDataByes),
				Foil:          cubeCobraCard.Finish == "Foil",
			}
			cfg.Cards = append(cfg.Cards, card)
		}
	}
	return cfg.Cards, nil
}
