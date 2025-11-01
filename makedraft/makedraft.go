package makedraft

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/objectbox/objectbox-go/objectbox"
	"github.com/walkingeyerobot/r38/draftconfig"
	"github.com/walkingeyerobot/r38/schema"
)

const GuildId = "685333271793500161"
const SpectatorsCategoryId = "711340302966980698"

var FakeSpectatorChannelID = ""

// Settings stores all the settings that can be passed in.
type Settings struct {
	Set                                       *string
	Database                                  *string
	DatabaseDir                               *string
	Seed                                      *int
	InPerson                                  *bool
	AssignSeats                               *bool
	AssignPacks                               *bool
	Verbose                                   *bool
	Simulate                                  *bool
	Name                                      *string
	MaxMythic                                 *int
	MaxRare                                   *int
	MaxUncommon                               *int
	MaxCommon                                 *int
	PackCommonColorStdevMax                   *float64
	PackCommonRatingMin                       *float64
	PackCommonRatingMax                       *float64
	DraftCommonColorStdevMax                  *float64
	PackCommonColorIdentityStdevMax           *float64
	DraftCommonColorIdentityStdevMax          *float64
	DfcMode                                   *bool
	AbortMissingCommonColor                   *bool
	AbortMissingCommonColorIdentity           *bool
	AbortDuplicateThreeColorIdentityUncommons *bool
	PickTwo                                   *bool
	UpdateExisting                            *uint64
}

func ParseSettings(args []string) (Settings, error) {
	flagSet := flag.NewFlagSet(args[0], flag.ContinueOnError)

	settings := Settings{}
	settings.Set = flagSet.String(
		"set", "sets/cube.json",
		"A .json file containing relevant set data.")
	settings.DatabaseDir = flagSet.String(
		"database_dir", "",
		"The objectbox database directory to insert to.")
	settings.Seed = flagSet.Int(
		"seed", 0,
		"The random seed to use to generate the draft. If 0, time.Now().UnixNano() will be used.")
	settings.InPerson = flagSet.Bool(
		"inPerson", false,
		"If true, draft will be initialized with empty packs.")
	settings.AssignSeats = flagSet.Bool(
		"assignSeats", false,
		"If true, players will be preassigned seats even for an in-person draft.")
	settings.AssignPacks = flagSet.Bool(
		"assignPacks", false,
		"If true, players will be preassigned their first packs even for an in-person draft.")
	settings.PickTwo = flagSet.Bool(
		"pickTwo", false,
		"If true, the created draft is a Pick Two draft (four players, two picks per pack).")
	settings.Verbose = flagSet.Bool(
		"v", false,
		"If true, will enable verbose output.")
	settings.Simulate = flagSet.Bool(
		"simulate", false,
		"If true, won't commit to the database.")
	settings.Name = flagSet.String(
		"name", "untitled draft",
		"The name of the draft.")
	settings.UpdateExisting = flagSet.Uint64(
		"updateExisting", 0,
		"If nonzero, updates cards in an existing draft rather than creating a new one.")

	err := flagSet.Parse(args[1:])

	return settings, err
}

// CardSet helps us lookup cards by rarity.
type CardSet struct {
	All       []draftconfig.Card
	Mythics   []draftconfig.Card
	Rares     []draftconfig.Card
	Uncommons []draftconfig.Card
	Commons   []draftconfig.Card
	Basics    []draftconfig.Card
}

func MakeDraft(settings Settings, ob *objectbox.ObjectBox) error {
	err := AddDraftConfigSettings(&settings)
	if err != nil {
		return err
	}

	random := getRNG(settings)

	if settings.UpdateExisting != nil && *settings.UpdateExisting > 0 {
		return UpdateDraft(settings, *settings.UpdateExisting, ob)
	}

	log.Printf("generating draft %s.", *settings.Name)

	var numPacks int
	if *settings.PickTwo {
		numPacks = 12
	} else {
		numPacks = 24
	}
	var cardsPerPack int
	if *settings.PickTwo {
		cardsPerPack = 14
	} else {
		cardsPerPack = 15
	}

	packs, err := GeneratePacks(settings)
	if err != nil {
		return err
	}

	format := path.Base(*settings.Set)
	format = strings.TrimSuffix(format, path.Ext(format))
	assignSeats := *settings.AssignSeats
	assignPacks := *settings.AssignPacks || !*settings.InPerson
	re := regexp.MustCompile(`"FOIL_STATUS"`)

	var numSeats int
	if *settings.PickTwo {
		numSeats = 4
	} else {
		numSeats = 8
	}
	var numUsers int
	if assignSeats {
		numUsers = numSeats
	} else {
		numUsers = 0
	}
	assignedUsers, err := AssignSeats(ob, 0, numUsers)
	if err != nil {
		return err
	}
	scanSounds := random.Perm(numSeats)
	errorSounds := random.Perm(numSeats)

	var seats []*schema.Seat
	for i := 0; i < numSeats; i++ {
		if len(assignedUsers) > i {
			reservedUser, err := schema.BoxForUser(ob).Get(uint64(assignedUsers[i]))
			if err != nil {
				return err
			}
			seat := schema.Seat{
				Position:      i,
				Round:         1,
				ReservedUser:  reservedUser,
				ScanSound:     scanSounds[i],
				ErrorSound:    errorSounds[i],
				Packs:         []*schema.Pack{},
				OriginalPacks: []*schema.Pack{},
				PickedCards:   []*schema.Card{},
			}
			seats = append(seats, &seat)
		} else {
			seat := schema.Seat{
				Position:      i,
				Round:         1,
				ScanSound:     scanSounds[i],
				ErrorSound:    errorSounds[i],
				Packs:         []*schema.Pack{},
				OriginalPacks: []*schema.Pack{},
				PickedCards:   []*schema.Card{},
			}
			seats = append(seats, &seat)
		}
	}

	var obPacks []*schema.Pack
	for i := range numPacks {
		pack := packs[i]
		var obCards []*schema.Card
		for j := range cardsPerPack {
			card := pack[j]
			var data string
			if card.Foil {
				data = re.ReplaceAllString(card.Data, "true")
			} else {
				data = re.ReplaceAllString(card.Data, "false")
			}
			obCards = append(obCards, &schema.Card{
				Data:   data,
				CardId: card.ID,
			})
		}
		obPack := schema.Pack{
			Round:         0,
			OriginalCards: obCards,
			Cards:         obCards,
		}
		obPacks = append(obPacks, &obPack)
	}

	if assignPacks {
		randPacks := random.Perm(len(obPacks))
		for i, seat := range seats {
			for j := range 3 {
				pack := obPacks[randPacks[i*3+j]]
				pack.Round = j + 1
				seat.Packs = append(seat.Packs, pack)
				seat.OriginalPacks = append(seat.Packs, pack)
			}
		}
		obPacks = []*schema.Pack{}
	}

	var dg *discordgo.Session
	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if !*settings.Simulate && len(botToken) > 0 {
		dg, err = discordgo.New("Bot " + botToken)
		if err != nil {
			return fmt.Errorf("error creating spectator channel: %v", err)
		}
	} else {
		dg = nil
	}
	var channel *discordgo.Channel
	var channelID string
	if dg != nil {
		channel, err = dg.GuildChannelCreate(GuildId,
			regexp.MustCompile("[^a-z0-9-]").ReplaceAllString(strings.ToLower(*settings.Name), "-")+"-spectators",
			discordgo.ChannelTypeGuildText)
		if err != nil {
			return fmt.Errorf("error creating spectator channel: %v", err)
		}
		channelID = channel.ID
	} else if FakeSpectatorChannelID != "" {
		channelID = FakeSpectatorChannelID
	} else {
		channelID = ""
	}

	draft := schema.Draft{
		Name:               *settings.Name,
		Format:             format,
		InPerson:           *settings.InPerson,
		Seats:              seats,
		UnassignedPacks:    obPacks,
		Events:             []*schema.Event{},
		SpectatorChannelId: channelID,
		PickTwo:            *settings.PickTwo,
	}

	draftId, err := schema.BoxForDraft(ob).Put(&draft)
	if err != nil {
		return err
	}

	if dg != nil {
		_, err = dg.ChannelEditComplex(channel.ID, &discordgo.ChannelEdit{
			Topic:    fmt.Sprintf("<https://draftcu.be/draft/%d>", draftId),
			ParentID: SpectatorsCategoryId,
		})
		if err != nil {
			return fmt.Errorf("error creating spectator channel: %v", err)
		}
		err = dg.Close()
		if err != nil {
			log.Printf("error closing bot session: %s", err)
		}
	}

	return nil
}

func getRNG(settings Settings) *rand.Rand {
	var random *rand.Rand
	if *settings.Seed == 0 {
		random = rand.New(rand.NewSource(time.Now().UnixNano()))
	} else {
		random = rand.New(rand.NewSource(int64(*settings.Seed)))
	}
	return random
}

func AddDraftConfigSettings(settings *Settings) error {
	flagSet := flag.FlagSet{}
	settings.MaxMythic = flagSet.Int(
		"max-mythic", 2,
		"Maximum number of copies of a given mythic allowed in a draft. 0 to disable.")
	settings.MaxRare = flagSet.Int(
		"max-rare", 3,
		"Maximum number of copies of a given rare allowed in a draft. 0 to disable.")
	settings.MaxUncommon = flagSet.Int(
		"max-uncommon", 4,
		"Maximum number of copies of a given uncommon allowed in a draft. 0 to disable.")
	settings.MaxCommon = flagSet.Int(
		"max-common", 6,
		"Maximum number of copies of a given common allowed in a draft. 0 to disable.")
	settings.PackCommonColorStdevMax = flagSet.Float64(
		"pack-common-color-stdev-max", 0,
		"Maximum standard deviation allowed in a pack of color distribution among commons. 0 to disable.")
	settings.PackCommonRatingMin = flagSet.Float64(
		"pack-common-rating-min", 0,
		"Minimum average rating allowed in a pack among commons. 0 to disable.")
	settings.PackCommonRatingMax = flagSet.Float64(
		"pack-common-rating-max", 0,
		"Maximum average rating allowed in a pack among commons. 0 to disable.")
	settings.DraftCommonColorStdevMax = flagSet.Float64(
		"draft-common-color-stdev-max", 0,
		"Maximum standard deviation allowed in the entire draft of color distribution among commons. 0 to disable.")
	settings.PackCommonColorIdentityStdevMax = flagSet.Float64(
		"pack-common-color-identity-stdev-max", 0,
		"Maximum standard deviation allowed in a pack of color identity distribution among commons. 0 to disable.")
	settings.DraftCommonColorIdentityStdevMax = flagSet.Float64(
		"draft-common-color-identity-stdev-max", 0,
		"Maximum standard deviation allowed in the entire draft of color identity distribution among commons. 0 to disable.")
	settings.DfcMode = flagSet.Bool(
		"dfc-mode", false,
		"If true, include DFCs only in DFC specific hoppers and exclude them from color distribution stats.")
	settings.AbortMissingCommonColor = flagSet.Bool(
		"abort-missing-common-color", false,
		"If true, every color will be represented in the colors of commons in every pack.")
	settings.AbortMissingCommonColorIdentity = flagSet.Bool(
		"abort-missing-common-color-identity", false,
		"If true, every color will be represented in the color identities of commons in every pack.")
	settings.AbortDuplicateThreeColorIdentityUncommons = flagSet.Bool(
		"abort-duplicate-three-color-identity-uncommons", false,
		"If true, only one uncommon of a color identity triplet will be allowed per pack.")

	cfg, err := getDraftConfig(*settings)
	if err != nil {
		return err
	}
	if len(cfg.Flags) != 0 {
		jsonFlags := strings.Join(cfg.Flags, " ")
		r := csv.NewReader(strings.NewReader(jsonFlags))
		r.Comma = ' '
		fields, err := r.Read()
		if err != nil {
			return fmt.Errorf("error parsing json flags: %w", err)
		}
		var allFlags []string
		for _, field := range fields {
			if field != "" {
				allFlags = append(allFlags, field)
			}
		}
		err = flagSet.Parse(allFlags)
		if err != nil {
			return fmt.Errorf("error parsing json flags: %w", err)
		}
	}
	return nil
}

func getDraftConfig(settings Settings) (draftconfig.DraftConfig, error) {
	if *settings.Set == "" {
		return draftconfig.DraftConfig{}, fmt.Errorf("you must specify a set json file to continue")
	}

	jsonFile, err := os.Open(*settings.Set)
	if err != nil {
		return draftconfig.DraftConfig{}, fmt.Errorf("error opening json file: %w", err)
	}
	defer func() {
		_ = jsonFile.Close()
	}()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return draftconfig.DraftConfig{}, fmt.Errorf("error readalling: %w", err)
	}

	var cfg draftconfig.DraftConfig
	err = json.Unmarshal(byteValue, &cfg)
	if err != nil {
		return draftconfig.DraftConfig{}, fmt.Errorf("error unmarshalling: %w", err)
	}
	return cfg, nil
}

func GeneratePacks(settings Settings) ([24][15]draftconfig.Card, error) {
	cfg, err := getDraftConfig(settings)
	if err != nil {
		return [24][15]draftconfig.Card{}, fmt.Errorf("error reading draft config: %w", err)
	}

	random := getRNG(settings)

	var numPacks int
	if *settings.PickTwo {
		numPacks = 12
	} else {
		numPacks = 24
	}
	var cardsPerPack int
	if *settings.PickTwo {
		cardsPerPack = 14
	} else {
		cardsPerPack = 15
	}

	var allCards CardSet
	var dfcCards CardSet

	configCards, err := draftconfig.GetCards(cfg)
	if err != nil {
		return [24][15]draftconfig.Card{}, fmt.Errorf("error getting cards: %w", err)
	}
	for _, card := range configCards {
		var currentSet *CardSet
		if *settings.DfcMode && card.Dfc {
			currentSet = &dfcCards
		} else {
			currentSet = &allCards
		}
		currentSet.All = append(currentSet.All, card)

		switch card.Rarity {
		case "mythic":
		case "bonus":
		case "special":
			currentSet.Mythics = append(currentSet.Mythics, card)
		case "rare":
			currentSet.Rares = append(currentSet.Rares, card)
		case "uncommon":
			currentSet.Uncommons = append(currentSet.Uncommons, card)
		case "common":
			currentSet.Commons = append(currentSet.Commons, card)
		case "basic":
			currentSet.Basics = append(currentSet.Basics, card)
		default:
			return [24][15]draftconfig.Card{}, fmt.Errorf("error with determining rarity for %v", card)
		}
	}

	var hoppers [15]draftconfig.Hopper
	resetHoppers := func() {
		for i, hopdef := range cfg.Hoppers {
			switch hopdef.Type {
			case "RareHopper":
				hoppers[i] = draftconfig.MakeNormalHopper(false, random, allCards.Mythics, allCards.Rares, allCards.Rares)
			case "RareRefillHopper":
				hoppers[i] = draftconfig.MakeNormalHopper(true, random, allCards.Mythics, allCards.Rares, allCards.Rares)
			case "UncommonHopper":
				hoppers[i] = draftconfig.MakeNormalHopper(false, random, allCards.Uncommons, allCards.Uncommons)
			case "UncommonRefillHopper":
				hoppers[i] = draftconfig.MakeNormalHopper(true, random, allCards.Uncommons, allCards.Uncommons)
			case "CommonHopper":
				hoppers[i] = draftconfig.MakeNormalHopper(false, random, allCards.Commons, allCards.Commons)
			case "CommonRefillHopper":
				hoppers[i] = draftconfig.MakeNormalHopper(true, random, allCards.Commons, allCards.Commons)
			case "BasicLandHopper":
				hoppers[i] = draftconfig.MakeBasicLandHopper(random, allCards.Basics)
			case "CubeHopper":
				hoppers[i] = draftconfig.MakeNormalHopper(false, random, allCards.All)
			case "Pointer":
				hoppers[i] = hoppers[hopdef.Refs[0]]
			case "DfcHopper":
				hoppers[i] = draftconfig.MakeNormalHopper(false,
					random,
					dfcCards.Mythics,
					dfcCards.Rares, dfcCards.Rares,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons)
			case "DfcRefillHopper":
				hoppers[i] = draftconfig.MakeNormalHopper(true,
					random,
					dfcCards.Mythics,
					dfcCards.Rares, dfcCards.Rares,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons)
			case "FoilHopper":
				hoppers[i] = draftconfig.MakeFoilHopper(&hoppers[hopdef.Refs[0]], &hoppers[hopdef.Refs[1]], &hoppers[hopdef.Refs[2]],
					random,
					allCards.Mythics,
					allCards.Rares, allCards.Rares,
					allCards.Uncommons, allCards.Uncommons, allCards.Uncommons,
					allCards.Commons, allCards.Commons, allCards.Commons, allCards.Commons,
					allCards.Basics, allCards.Basics, allCards.Basics, allCards.Basics)
			}
		}
	}

	var packs [24][15]draftconfig.Card
	packAttempts := 0
	draftAttempts := 0

	for {
		resetHoppers()
		resetDraft := false
		draftAttempts++
		for i := 0; i < numPacks; { // we'll manually increment i
			packAttempts++
			for j := range cardsPerPack {
				var empty bool
				packs[i][j], empty = hoppers[j].Pop(random)
				if empty {
					resetDraft = true
					break
				}
			}

			if resetDraft {
				break
			}

			if *settings.Verbose {
				for j := range cardsPerPack {
					log.Printf("%s\t%v\t%s", packs[i][j].Rarity, packs[i][j].Foil, packs[i][j].Data)
				}
			}

			if okPack(packs[i], settings) {
				i++
			}
		}
		if !resetDraft && (okDraft(packs, settings)) {
			break
		}

		if *settings.Verbose {
			log.Printf("RESETTING DRAFT")
		}
	}

	if *settings.Verbose {
		log.Printf("draft attempts: %d", draftAttempts)
		log.Printf("pack attempts: %d", packAttempts)
	}
	return packs, nil
}

func okPack(pack [15]draftconfig.Card, settings Settings) bool {
	passes := true
	cardHash := make(map[string]int)
	colorHash := make(map[rune]float64)
	colorIdentityHash := make(map[rune]float64)
	uncommonColorIdentities := make(map[string]int)
	var ratings []float64
	totalCommons := 0
	var cardsPerPack int
	if *settings.PickTwo {
		cardsPerPack = 14
	} else {
		cardsPerPack = 15
	}
	for i := range cardsPerPack {
		card := pack[i]
		if *settings.Verbose {
			log.Printf("%v %v", card, settings.DfcMode)
		}
		if card.Foil || (*settings.DfcMode && card.Dfc) {
			continue
		}
		cardHash[card.ID]++
		if cardHash[card.ID] > 1 {
			if *settings.Verbose {
				log.Printf("found duplicated card %s", card.ID)
			}
			passes = false
		}
		if card.Rarity == "common" {
			for _, color := range card.Color {
				colorHash[color]++
			}
			for _, color := range card.ColorIdentity {
				colorIdentityHash[color]++
			}
			ratings = append(ratings, card.Rating)
			totalCommons++
		} else if card.Rarity == "uncommon" {
			if *settings.AbortDuplicateThreeColorIdentityUncommons {
				sortedColor := stringSort(card.ColorIdentity)
				uncommonColorIdentities[sortedColor]++
				if uncommonColorIdentities[sortedColor] > 1 && len(sortedColor) == 3 {
					if *settings.Verbose {
						log.Printf("found more than one uncommon %s card", sortedColor)
					}
					passes = false
				}
				if uncommonColorIdentities[sortedColor] >= 3 {
					if *settings.Verbose {
						log.Printf("all uncommons are %s", sortedColor)
					}
					passes = false
				}
			}
		}
	}

	// calculate stdev for color
	var colors []float64
	for _, v := range colorHash {
		colors = append(colors, v)
	}

	if *settings.AbortMissingCommonColor && len(colors) != 5 {
		if *settings.Verbose {
			log.Printf("a color is missing among commons")
		}
		passes = false
		for {
			colors = append(colors, 0)
			if len(colors) == 5 {
				break
			}
		}
	}

	colorStdev := stdev(colors)

	var colorIdentities []float64
	for _, v := range colorIdentityHash {
		colorIdentities = append(colorIdentities, v)
	}

	if *settings.AbortMissingCommonColorIdentity && len(colorIdentities) != 5 {
		if *settings.Verbose {
			log.Printf("a color identity is missing among commons")
		}
		passes = false
		for {
			colorIdentities = append(colorIdentities, 0)
			if len(colorIdentities) == 5 {
				break
			}
		}
	}

	colorIdentityStdev := stdev(colorIdentities)

	ratingMean := mean(ratings)
	if *settings.Verbose {
		log.Printf("color stdev:\t%f", colorStdev)
		log.Printf("color identity stdev:\t%f", colorIdentityStdev)
		log.Printf("rating mean:\t%f", ratingMean)
	}

	if *settings.PackCommonColorStdevMax != 0 && colorStdev > *settings.PackCommonColorStdevMax {
		if *settings.Verbose {
			log.Printf("color stdev too high")
		}
		passes = false
	}
	if *settings.PackCommonColorIdentityStdevMax != 0 && colorIdentityStdev > *settings.PackCommonColorIdentityStdevMax {
		if *settings.Verbose {
			log.Printf("color identity stdev too high")
		}
		passes = false
	}
	if *settings.PackCommonRatingMax != 0 && ratingMean > *settings.PackCommonRatingMax {
		if *settings.Verbose {
			log.Printf("rating mean too high")
		}
		passes = false
	} else if *settings.PackCommonRatingMin != 0 && ratingMean < *settings.PackCommonRatingMin {
		if *settings.Verbose {
			log.Printf("rating mean too low")
		}
		passes = false
	}

	if passes {
		if *settings.Verbose {
			log.Printf("pack passes!")
		}
	} else if *settings.Verbose {
		log.Printf("pack fails :(")
	}

	return passes
}

func okDraft(packs [24][15]draftconfig.Card, settings Settings) bool {
	if *settings.Verbose {
		log.Printf("analyzing entire draft pool...")
	}
	passes := true

	cardHash := make(map[string]int)
	colorHash := make(map[rune]float64)
	colorIdentityHash := make(map[rune]float64)
	q := make(map[string]int)
	var numPacks int
	var cardsPerPack int
	if *settings.PickTwo {
		numPacks = 12
		cardsPerPack = 14
	} else {
		numPacks = 24
		cardsPerPack = 15
	}
	for i := range numPacks {
		pack := packs[i]
		for j := range cardsPerPack {
			card := pack[j]
			cardHash[card.ID]++
			qty := cardHash[card.ID]
			if card.Rarity != "B" && qty > q[card.Rarity] {
				q[card.Rarity] = qty
			}
			switch card.Rarity {
			case "mythic":
				if *settings.MaxMythic != 0 && qty > *settings.MaxMythic {
					if *settings.Verbose {
						log.Printf("found %d %s, which is more than %d", qty, card.ID, *settings.MaxMythic)
					}
					passes = false
				}
			case "rare":
				if *settings.MaxRare != 0 && qty > *settings.MaxRare {
					if *settings.Verbose {
						log.Printf("found %d %s, which is more than %d", qty, card.ID, *settings.MaxRare)
					}
					passes = false
				}
			case "uncommon":
				if *settings.MaxUncommon != 0 && qty > *settings.MaxUncommon {
					if *settings.Verbose {
						log.Printf("found %d %s, which is more than %d", qty, card.ID, *settings.MaxUncommon)
					}
					passes = false
				}
			case "common":
				if *settings.MaxCommon != 0 && qty > *settings.MaxCommon {
					if *settings.Verbose {
						log.Printf("found %d %s, which is more than %d", qty, card.ID, *settings.MaxCommon)
					}
					passes = false
				}
				if !(card.Foil || (*settings.DfcMode && card.Dfc)) {
					for _, color := range card.Color {
						colorHash[color]++
					}
					for _, color := range card.ColorIdentity {
						colorIdentityHash[color]++
					}
				}
			}
		}
	}

	// calculate stdev for color
	var colors []float64
	for _, v := range colorHash {
		colors = append(colors, v)
	}

	colorStdev := stdev(colors)

	var colorIdentities []float64
	for _, v := range colorIdentityHash {
		colorIdentities = append(colorIdentities, v)
	}

	colorIdentityStdev := stdev(colorIdentities)

	if *settings.Verbose {
		log.Printf("all commons color stdev:\t%f\t%v", colorStdev, colorHash)
		log.Printf("all commons color identity stdev:\t%f\t%v", colorIdentityStdev, colorIdentityHash)
	}

	if *settings.DraftCommonColorStdevMax != 0 && colorStdev > *settings.DraftCommonColorStdevMax {
		if *settings.Verbose {
			log.Printf("color stdev too high")
		}
		passes = false
	}

	if *settings.DraftCommonColorIdentityStdevMax != 0 && colorIdentityStdev > *settings.DraftCommonColorIdentityStdevMax {
		if *settings.Verbose {
			log.Printf("color identity stdev too high")
		}
		passes = false
	}

	if passes {
		if *settings.Verbose {
			log.Printf("draft passes!")
		}
	} else if *settings.Verbose {
		log.Printf("draft fails :(")
	}

	return passes
}

func AssignSeats(ob *objectbox.ObjectBox, draftId int64, numUsers int) ([]int, error) {
	var userIds []int
	if numUsers == 0 {
		return userIds, nil
	}

	if draftId != 0 {
		draft, err := schema.BoxForDraft(ob).Get(uint64(draftId))
		if err != nil {
			return userIds, err
		}

		users, err := schema.BoxForUser(ob).GetAll() // TODO: need to limit?
		if err != nil {
			return userIds, err
		}
		candidates := rand.Perm(len(users))
		for _, i := range candidates {
			if slices.IndexFunc(users[i].Skips, func(skip *schema.Skip) bool {
				return skip.DraftId == uint64(draftId)
			}) != -1 {
				continue
			}
			userId := users[i].Id
			if slices.IndexFunc(draft.Seats, func(seat *schema.Seat) bool {
				return seat.User.Id == userId || seat.ReservedUser.Id == userId
			}) != -1 {
				continue
			}
			userIds = append(userIds, int(userId))
			if len(userIds) == numUsers {
				break
			}
		}
	} else {
		users, err := schema.BoxForUser(ob).GetAll() // TODO: need to limit?
		if err != nil {
			return userIds, err
		}
		candidates := rand.Perm(len(users))
		for _, i := range candidates {
			userIds = append(userIds, int(users[i].Id))
		}
	}

	return userIds, nil
}

func UpdateDraft(settings Settings, draftID uint64, ob *objectbox.ObjectBox) error {
	return ob.RunInWriteTx(func() error {
		cfg, err := getDraftConfig(settings)

		draft, err := schema.BoxForDraft(ob).Get(draftID)
		if err != nil {
			return err
		}

		cardBox := schema.BoxForCard(ob)

		cards, err := draftconfig.GetCards(cfg)
		if err != nil {
			return err
		}

		cardsMap := make(map[string]draftconfig.Card)
		for _, card := range cards {
			log.Printf("building card map: %s %s", card.ID, card.Data)
			cardsMap[card.ID] = card
		}

		for _, seat := range draft.Seats {
			for _, card := range seat.PickedCards {
				err = updateCard(cardsMap, card, cardBox)
				if err != nil {
					return err
				}
			}
			for _, pack := range seat.Packs {
				for _, card := range pack.Cards {
					err = updateCard(cardsMap, card, cardBox)
					if err != nil {
						return err
					}
				}
			}
		}
		for _, pack := range draft.UnassignedPacks {
			for _, card := range pack.Cards {
				err = updateCard(cardsMap, card, cardBox)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func updateCard(cardsMap map[string]draftconfig.Card, card *schema.Card, cardBox *schema.CardBox) error {
	newCard := cardsMap[card.CardId]
	if len(newCard.Data) > 0 {
		log.Printf("updating card %s: %s", card.CardId, newCard.Data)
		card.Data = newCard.Data
		_, err := cardBox.Put(card)
		if err != nil {
			return err
		}
	} else {
		log.Printf("didn't find card %s", card.CardId)
	}
	return nil
}

func stdev(list []float64) float64 {
	avg := mean(list)

	var sum float64
	for _, val := range list {
		sum += math.Pow(val-avg, 2)
	}
	return math.Sqrt(sum / float64(len(list)))
}

func mean(list []float64) float64 {
	var sum float64
	for _, val := range list {
		sum += val
	}
	return sum / float64(len(list))
}

// from https://stackoverflow.com/questions/22688651/golang-how-to-sort-string-or-byte
// go generics when
type sortRunes []rune

func (s sortRunes) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s sortRunes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortRunes) Len() int {
	return len(s)
}

func stringSort(s string) string {
	r := []rune(s)
	sort.Sort(sortRunes(r))
	return string(r)
}
