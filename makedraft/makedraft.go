package makedraft

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

const GUILD_ID = "685333271793500161"
const SPECTATORS_CATEGORY_ID = "711340302966980698"

// Settings stores all the settings that can be passed in.
type Settings struct {
	Set                                       *string
	Database                                  *string
	Seed                                      *int
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
}

func MakeDraft(settings Settings, tx *sql.Tx) error {
	if *settings.Set == "" {
		return fmt.Errorf("you must specify a set json file to continue")
	}

	jsonFile, err := os.Open(*settings.Set)
	if err != nil {
		return fmt.Errorf("error opening json file: %s", err.Error())
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return fmt.Errorf("error readalling: %s", err.Error())
	}

	var cfg DraftConfig
	err = json.Unmarshal(byteValue, &cfg)
	if err != nil {
		return fmt.Errorf("error unmarshalling: %s", err.Error())
	}

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

	if len(cfg.Flags) != 0 {
		jsonFlags := strings.Join(cfg.Flags, " ")
		r := csv.NewReader(strings.NewReader(jsonFlags))
		r.Comma = ' '
		fields, err := r.Read()
		if err != nil {
			return fmt.Errorf("error parsing json flags: %s", err.Error())
		}
		var allFlags []string
		for _, flag := range fields {
			if flag != "" {
				allFlags = append(allFlags, flag)
			}
		}
		flagSet.Parse(allFlags)
	}

	if *settings.Seed == 0 {
		rand.Seed(time.Now().UnixNano())
	} else {
		rand.Seed(int64(*settings.Seed))
	}

	log.Printf("generating draft %s.", *settings.Name)

	var packIDs [24]int64

	var allCards CardSet
	var dfcCards CardSet

	for _, card := range cfg.Cards {
		var currentSet *CardSet
		if *settings.DfcMode && card.Dfc {
			currentSet = &dfcCards
		} else {
			currentSet = &allCards
		}
		currentSet.All = append(currentSet.All, card)

		switch card.Rarity {
		case "mythic":
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
			return fmt.Errorf("error with determining rarity for %v", card)
		}
	}

	var hoppers [15]Hopper
	resetHoppers := func() {
		for i, hopdef := range cfg.Hoppers {
			switch hopdef.Type {
			case "RareHopper":
				hoppers[i] = MakeNormalHopper(false, allCards.Mythics, allCards.Rares, allCards.Rares)
			case "RareRefillHopper":
				hoppers[i] = MakeNormalHopper(true, allCards.Mythics, allCards.Rares, allCards.Rares)
			case "UncommonHopper":
				hoppers[i] = MakeNormalHopper(false, allCards.Uncommons, allCards.Uncommons)
			case "UncommonRefillHopper":
				hoppers[i] = MakeNormalHopper(true, allCards.Uncommons, allCards.Uncommons)
			case "CommonHopper":
				hoppers[i] = MakeNormalHopper(false, allCards.Commons, allCards.Commons)
			case "CommonRefillHopper":
				hoppers[i] = MakeNormalHopper(true, allCards.Commons, allCards.Commons)
			case "BasicLandHopper":
				hoppers[i] = MakeBasicLandHopper(allCards.Basics)
			case "CubeHopper":
				hoppers[i] = MakeNormalHopper(false, allCards.All)
			case "Pointer":
				hoppers[i] = hoppers[hopdef.Refs[0]]
			case "DfcHopper":
				hoppers[i] = MakeNormalHopper(false,
					dfcCards.Mythics,
					dfcCards.Rares, dfcCards.Rares,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons)
			case "DfcRefillHopper":
				hoppers[i] = MakeNormalHopper(true,
					dfcCards.Mythics,
					dfcCards.Rares, dfcCards.Rares,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Uncommons, dfcCards.Uncommons, dfcCards.Uncommons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons, dfcCards.Commons,
					dfcCards.Commons, dfcCards.Commons, dfcCards.Commons)
			case "FoilHopper":
				hoppers[i] = MakeFoilHopper(&hoppers[hopdef.Refs[0]], &hoppers[hopdef.Refs[1]], &hoppers[hopdef.Refs[2]],
					allCards.Mythics,
					allCards.Rares, allCards.Rares,
					allCards.Uncommons, allCards.Uncommons, allCards.Uncommons,
					allCards.Commons, allCards.Commons, allCards.Commons, allCards.Commons,
					allCards.Basics, allCards.Basics, allCards.Basics, allCards.Basics)
			}
		}
	}

	var packs [24][15]Card
	packAttempts := 0
	draftAttempts := 0

	for {
		resetHoppers()
		resetDraft := false
		draftAttempts++
		for i := 0; i < 24; { // we'll manually increment i
			packAttempts++
			for j, hopper := range hoppers {
				var empty bool
				packs[i][j], empty = hopper.Pop()
				if empty {
					resetDraft = true
					break
				}
			}

			if resetDraft {
				break
			}

			if *settings.Verbose {
				for _, card := range packs[i] {
					log.Printf("%s\t%v\t%s", card.Rarity, card.Foil, card.Data)
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

	format := path.Base(*settings.Set)
	format = strings.TrimSuffix(format, path.Ext(format))
	packIDs, err = generateEmptyDraft(tx, *settings.Name, format, *settings.Simulate)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(`"FOIL_STATUS"`)
	if *settings.Verbose {
		log.Printf("inserting into db...")
	}
	query := `INSERT INTO cards (pack, original_pack, data) VALUES (?, ?, ?)`
	for i, pack := range packs {
		for _, card := range pack {
			packID := packIDs[i]
			var data string
			if card.Foil {
				data = re.ReplaceAllString(card.Data, "true")
			} else {
				data = re.ReplaceAllString(card.Data, "false")
			}
			tx.Exec(query, packID, packID, data)
		}
	}
	return nil
}

func generateEmptyDraft(tx *sql.Tx, name string, format string, simulate bool) ([24]int64, error) {
	var packIds [24]int64

	var dg *discordgo.Session
	var err error
	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if !simulate && len(botToken) > 0 {
		dg, err = discordgo.New("Bot " + botToken)
		if err != nil {
			return packIds, fmt.Errorf("error creating spectator channel: %s", err)
		}
	} else {
		dg = nil
	}

	var channel *discordgo.Channel
	var channelID string
	if dg != nil {
		channel, err = dg.GuildChannelCreate(GUILD_ID,
			regexp.MustCompile("[^a-z-]").ReplaceAllString(strings.ToLower(name), "-") + "-spectators",
			discordgo.ChannelTypeGuildText)
		if err != nil {
			return packIds, fmt.Errorf("error creating spectator channel: %s", err)
		}
		channelID = channel.ID
	} else {
		channelID = ""
	}

	query := `INSERT INTO drafts (name, spectatorchannelid, format) VALUES (?, ?, ?);`
	res, err := tx.Exec(query, name, channelID, format)
	if err != nil {
		return packIds, fmt.Errorf("error creating draft: %s", err)
	}

	draftID, err := res.LastInsertId()
	if err != nil {
		return packIds, fmt.Errorf("could not get draft ID: %s", err)
	}

	if dg != nil {
		_, err = dg.ChannelEditComplex(channel.ID, &discordgo.ChannelEdit{
			Topic: fmt.Sprintf("<http://draft.thefoley.net/draft/%d>", draftID),
			ParentID: SPECTATORS_CATEGORY_ID,
		})
		if err != nil {
			return packIds, fmt.Errorf("error creating spectator channel: %s", err)
		}
		err = dg.Close()
		if err != nil {
			log.Printf("error closing bot session: %s", err)
		}
	}

	assignedUsers, err := AssignSeats(tx, draftID, format, 8)
	if err != nil {
		return packIds, fmt.Errorf("error assigning users: %s", err.Error())
	}
	var seatIds [8]int64
	for i := 0; i < 8; i++ {
		if len(assignedUsers) > i {
			query = `INSERT INTO seats (position, draft, reserveduser) VALUES (?, ?, ?)`
			res, err = tx.Exec(query, i, draftID, assignedUsers[i])
		} else {
			query = `INSERT INTO seats (position, draft) VALUES (?, ?)`
			res, err = tx.Exec(query, i, draftID)
		}
		if err != nil {
			return packIds, fmt.Errorf("could not create seats in draft: %s", err)
		}
		seatIds[i], err = res.LastInsertId()
		if err != nil {
			return packIds, fmt.Errorf("could not finalize seat creation: %s", err)
		}
	}

	query = `INSERT INTO packs (seat, original_seat, round) VALUES (?, ?, ?)`
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			res, err = tx.Exec(query, seatIds[i], seatIds[i], j)
			if err != nil {
				return packIds, fmt.Errorf("error creating packs: %s", err)
			}
			if j != 0 {
				packIds[(3*i)+(j-1)], err = res.LastInsertId()
				if err != nil {
					return packIds, fmt.Errorf("error creating packs: %s", err)
				}
			}
		}
	}

	return packIds, nil
}

func okPack(pack [15]Card, settings Settings) bool {
	passes := true
	cardHash := make(map[string]int)
	colorHash := make(map[rune]float64)
	colorIdentityHash := make(map[rune]float64)
	uncommonColorIdentities := make(map[string]int)
	var ratings []float64
	totalCommons := 0
	for _, card := range pack {
		log.Printf("%v %v", card, settings.DfcMode)
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

func okDraft(packs [24][15]Card, settings Settings) bool {
	if *settings.Verbose {
		log.Printf("analyzing entire draft pool...")
	}
	passes := true

	cardHash := make(map[string]int)
	colorHash := make(map[rune]float64)
	colorIdentityHash := make(map[rune]float64)
	q := make(map[string]int)
	for _, pack := range packs {
		for _, card := range pack {
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

func AssignSeats(tx *sql.Tx, draftID int64, format string, numUsers int) ([]int, error) {
	var minEpoch int
	var maxEpoch int
	query := `SELECT
				min(epoch), max(epoch)
				from userformats
				where elig = 1 and format = ?`
	row := tx.QueryRow(query, format)
	err := row.Scan(&minEpoch, &maxEpoch)
	if err != nil {
		return nil, err
	}
	var draftEpoch int
	if minEpoch == maxEpoch {
		draftEpoch = maxEpoch + 1
	} else {
		draftEpoch = maxEpoch
	}

	query = `select
				userformats.user
				from userformats
				left outer join skips on userformats.user = skips.user and skips.draft = ?
				left outer join seats 
				    on (userformats.user = seats.reserveduser or userformats.user = seats.user)
						and seats.draft = ?
				where elig = 1 and format = ? and skips.id is null and seats.id is null
				order by epoch, random()
				limit ?`
	result, err := tx.Query(query, draftID, draftID, format, numUsers)
	if err != nil {
		return nil, err
	}
	var users []int
	for result.Next() {
		var user int
		result.Scan(&user)
		users = append(users, user)

		query = `UPDATE userformats SET epoch = max(?, epoch + 1) where user = ? and format = ?`
		_, err = tx.Exec(query, draftEpoch, user, format)
	}

	return users, nil
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
