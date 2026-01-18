package htmajong

import (
	"math/rand"
	"time"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// DeckGenerator 会同麻将牌局生成器 (108张牌)
type DeckGenerator struct {
	rand *rand.Rand
}

// NewDeckGenerator 创建牌局生成器
func NewDeckGenerator() *DeckGenerator {
	return &DeckGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateDeck 生成牌堆 (108张: 万条筒各36张)
func (d *DeckGenerator) GenerateDeck() []core.Tile {
	tiles := make([]core.Tile, 0, 108)

	// 生成万、条、筒各36张 (1-9各4张)
	suits := []core.TileSuit{core.TileSuitWan, core.TileSuitTiao, core.TileSuitTong}

	for _, suit := range suits {
		for value := int8(1); value <= 9; value++ {
			for count := 0; count < 4; count++ {
				tiles = append(tiles, core.Tile{
					Suit:  suit,
					Value: value,
				})
			}
		}
	}

	return tiles
}

// Shuffle 洗牌
func (d *DeckGenerator) Shuffle(tiles []core.Tile) {
	d.rand.Shuffle(len(tiles), func(i, j int) {
		tiles[i], tiles[j] = tiles[j], tiles[i]
	})
}

// Deal 发牌 (每人13张,庄家14张)
func (d *DeckGenerator) Deal(tiles []core.Tile, playerCount int, dealerIndex int) (hands map[int][]core.Tile, remaining []core.Tile) {
	hands = make(map[int][]core.Tile)
	index := 0

	// 每人发13张
	for i := 0; i < playerCount; i++ {
		handSize := 13
		if i == dealerIndex {
			handSize = 14 // 庄家14张
		}

		hands[i] = tiles[index : index+handSize]
		index += handSize
	}

	// 剩余的牌
	remaining = tiles[index:]

	return hands, remaining
}
