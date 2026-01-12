package htmajong

// 麻将算法相关常量
const (
	// 标准手牌数量（不包括摸到的牌）
	StandardHandSize = 13
	// 完整手牌数量（包括摸到的牌）
	FullHandSize = 14
	// 碰牌数量
	PengCount = 3
	// 对子数量
	PairCount = 2
	// 杠类型常量
	GangTypeNone    = 0 // 不能杠
	GangTypeNormal  = 1 // 普通杠
	GangTypePublic  = 2 // 公杠（明杠）
	GangTypePrivate = 3 // 暗杠
)

// Algorithm 麻将算法类
type Algorithm struct{}

// ========== 辅助函数 ==========

// countMahjongNumber 统计手牌中指定数字的麻将数量
func countMahjongNumber(mahjongs []Mahjong, number int) int {
	count := 0
	for _, m := range mahjongs {
		if m.Number == number {
			count++
		}
	}
	return count
}

// buildNumberSlice 构建包含手牌和指定麻将的数字切片
func buildNumberSlice(seat *Seat, mahjong Mahjong, includePublic bool) []int {
	capacity := len(seat.ExtraList()) + 1
	if includePublic {
		capacity += len(seat.PublicList())
	}

	numbers := make([]int, 0, capacity)
	for _, m := range seat.ExtraList() {
		numbers = append(numbers, m.Number)
	}
	if includePublic {
		for _, m := range seat.PublicList() {
			numbers = append(numbers, m.Number)
		}
	}
	numbers = append(numbers, mahjong.Number)
	return numbers
}

// buildCountMap 构建麻将数字的计数映射
func buildCountMap(numbers []int) map[int]int {
	countMap := make(map[int]int, len(numbers))
	for _, num := range numbers {
		countMap[num]++
	}
	return countMap
}

// isTwoFiveEightNumber 判断数字是否是2、5、8
func isTwoFiveEightNumber(num int) bool {
	mod := num % 10
	return mod == 2 || mod == 5 || mod == 8
}

// ========== 公开方法 ==========

// CheckPublic 检查是否可以报听
func CheckPublic(table *Table, seat *Seat) bool {
	if seat.Step().Load() != 1 {
		return false
	}

	// 检查公开牌：碰了不能报听，但是杠可以报听
	if len(seat.PublicList()) > 0 {
		countMap := buildCountMap(mahjongSliceToNumbers(seat.PublicList()))
		for _, count := range countMap {
			if count == PengCount {
				return false
			}
		}
	}

	// 检查所有可能胡的牌
	mahjongList := Generate(1)
	publicWinList := make([]Mahjong, 0, 27) // 预分配容量（最多27种牌）
	for _, mahjong := range mahjongList {
		if CheckHuForPublic(CATCH, table.CurrentSeat, seat, mahjong) {
			publicWinList = append(publicWinList, mahjong)
		}
	}

	if len(publicWinList) > 0 {
		seat.SetPublicWinMahjong(publicWinList)
		return true
	}
	return false
}

// mahjongSliceToNumbers 将麻将切片转换为数字切片
func mahjongSliceToNumbers(mahjongs []Mahjong) []int {
	numbers := make([]int, len(mahjongs))
	for i, m := range mahjongs {
		numbers[i] = m.Number
	}
	return numbers
}

// CheckHUType 查看胡牌类型
func CheckHUType(table *Table, supplierType SupplierType, seat *Seat, mahjong Mahjong) []HuType {
	res := make([]HuType, 0)
	if supplierType == CATCH {
		commonCheckHu(table, supplierType, seat, mahjong, &res)
		if noJiang(supplierType, seat, mahjong) {
			res = res[:0]
			res = append(res, NO_JIANG)
		}
		if twoColor(supplierType, seat, mahjong) {
			res = res[:0]
			res = append(res, TWO_COLOR)
		}
	} else {
		commonCheckHu(table, supplierType, seat, mahjong, &res)
	}
	if len(res) == 0 {
		res = append(res, GENERAL)
	}
	return res
}

// commonCheckHu 通用胡牌检查
func commonCheckHu(table *Table, supplierType SupplierType, seat *Seat, mahjong Mahjong, res *[]HuType) {
	if checkClear(seat, mahjong) {
		*res = append(*res, CLEAR)
	}
	if checkPengPengHu(seat, mahjong) {
		*res = append(*res, PENG_PENG_HU)
	}
	if twoFiveEight(seat, mahjong) {
		*res = append(*res, TWO_FIVE_EIGHT)
	}
	if sevenPair(seat, mahjong) {
		*res = append(*res, SEVEN_PAIR)
	}
	if loongSevenPair(seat, mahjong) {
		*res = (*res)[:0]
		*res = append(*res, LOONG_SEVEN_PAIR)
	}
	if table.CanPublic && checkBaoTing(table, supplierType, seat) {
		*res = (*res)[:0]
		*res = append(*res, BAO_TING)
	}
}

// checkBaoTing 检查是否报听
func checkBaoTing(table *Table, supplierType SupplierType, seat *Seat) bool {
	if supplierType == CATCH {
		return seat.IsPublic()
	}
	if supplierType == OUT {
		return seat.IsPublic() || table.Lease.HappenedUser.IsPublic()
	}
	return false
}

// CheckPeng 检查是否可以碰
func CheckPeng(supplierType SupplierType, supplierUser *Seat, seat *Seat, mahjong Mahjong) bool {
	if supplierType != OUT || supplierUser == seat {
		return false
	}
	return countMahjongNumber(seat.ExtraList(), mahjong.Number) == PairCount
}

// CheckGang 检查是否可以杠
// 返回值：GangTypeNone(0)-不能杠，GangTypeNormal(1)-杠，GangTypePublic(2)-公杠，GangTypePrivate(3)-暗杠
func CheckGang(table *Table, supplierType SupplierType, supplierUser *Seat, seat *Seat, mahjong Mahjong) int {
	// 无牌不让杠
	if len(table.Extra) == 0 {
		return GangTypeNone
	}

	if supplierType == OUT {
		// 不能自己出牌自己杠
		if supplierUser == seat {
			return GangTypeNone
		}
		if countMahjongNumber(seat.ExtraList(), mahjong.Number) == PengCount {
			if !checkGangForPublic(table, seat, mahjong) {
				return GangTypeNone
			}
			return GangTypeNormal
		}
	} else if supplierType == CATCH {
		// 检查公杠
		if countMahjongNumber(seat.PublicList(), mahjong.Number) == PengCount {
			return GangTypePublic
		}
		// 检查暗杠
		if countMahjongNumber(seat.ExtraList(), mahjong.Number) == PengCount {
			if !checkGangForPublic(table, seat, mahjong) {
				return GangTypeNone
			}
			return GangTypePrivate
		}
	}
	return GangTypeNone
}

// checkGangForPublic 检查报听情况下杠牌是否改变牌型
func checkGangForPublic(table *Table, seat *Seat, mahjong Mahjong) bool {
	if !table.CanPublic {
		return true
	}
	// 检查是否报听以及是否改变牌型
	if seat.IsPublic() {
		// 创建一个临时座位用于测试
		copySeat := GenerateSeat(seat.User(), seat.Position())
		// 复制手牌，但排除杠的牌
		tempHand := make([]Mahjong, 0)
		for _, m := range seat.ExtraList() {
			if m.Number != mahjong.Number {
				tempHand = append(tempHand, m)
			}
		}
		// 设置手牌到临时座位
		copySeat.hand.SetTiles(tempHand)

		mahjongList := Generate(1)
		publicWinMap := make(map[int]bool)
		for _, mah := range mahjongList {
			if CheckHu(CATCH, seat, copySeat, mah) {
				publicWinMap[mah.Number] = true
			}
		}
		// 比较牌型是否相同
		compareMap := make(map[int]bool)
		for _, m := range seat.PublicWinMahjong() {
			compareMap[m.Number] = true
		}
		if len(compareMap) != len(publicWinMap) {
			return false
		}
		for k := range compareMap {
			if !publicWinMap[k] {
				return false
			}
		}
		return true
	}
	return true
}

// CheckHu 检查是否能胡
func CheckHu(supplierType SupplierType, supplierUser *Seat, seat *Seat, mahjong Mahjong) bool {
	// 不能抢杠胡自己
	if supplierType == GANG && supplierUser == seat {
		return false
	}
	return general(supplierType, supplierUser, seat, mahjong) ||
		sevenPair(seat, mahjong) ||
		twoFiveEight(seat, mahjong) ||
		twoColor(supplierType, seat, mahjong) ||
		noJiang(supplierType, seat, mahjong)
}

// CheckHuForPublic 检查是否能胡（用于报听）
func CheckHuForPublic(supplierType SupplierType, supplierUser *Seat, seat *Seat, mahjong Mahjong) bool {
	// 不能抢杠胡自己
	if supplierType == GANG && supplierUser == seat {
		return false
	}
	return general(supplierType, supplierUser, seat, mahjong) ||
		sevenPair(seat, mahjong) ||
		twoFiveEight(seat, mahjong)
}

// general 判断是否是通用胡牌牌型
func general(supplierType SupplierType, supplierUser *Seat, seat *Seat, mahjong Mahjong) bool {
	numbers := buildNumberSlice(seat, mahjong, false)
	countMap := buildCountMap(numbers)

	if !CanFormWinningHand(countMap, false) {
		return false
	}

	// 自摸跟抢杠胡 直接返回
	if supplierType == CATCH || supplierType == GANG {
		return true
	}

	// 检查牌型：清一色、碰碰胡 大胡 直接返回可以胡
	if checkClear(seat, mahjong) || checkPengPengHu(seat, mahjong) {
		return true
	}

	// 平胡：检查是否有报听 或者出牌的人是否报听
	return seat.IsPublic() || supplierUser.IsPublic()
}

// checkClear 检查牌型是否是清一色
func checkClear(seat *Seat, mahjong Mahjong) bool {
	// 快速检查：如果手牌和公开牌都为空则无法判断
	if len(seat.ExtraList()) == 0 && len(seat.PublicList()) == 0 {
		return false
	}

	// 获取第一张牌的颜色（通过除以10）
	var firstColor int
	if len(seat.ExtraList()) > 0 {
		firstColor = seat.ExtraList()[0].Number / 10
	} else {
		firstColor = seat.PublicList()[0].Number / 10
	}

	// 检查所有牌是否同一颜色
	checkColor := func(num int) bool {
		return num/10 == firstColor
	}

	for _, m := range seat.ExtraList() {
		if !checkColor(m.Number) {
			return false
		}
	}
	for _, m := range seat.PublicList() {
		if !checkColor(m.Number) {
			return false
		}
	}
	return checkColor(mahjong.Number)
}

// checkPengPengHu 检查牌型是否是碰碰胡
func checkPengPengHu(seat *Seat, mahjong Mahjong) bool {
	numbers := buildNumberSlice(seat, mahjong, true)
	countMap := buildCountMap(numbers)

	pairCount := 0
	for _, count := range countMap {
		if count == PairCount {
			pairCount++
			if pairCount > 1 {
				return false
			}
		} else if count == 1 {
			return false
		}
	}
	return true
}

// CanFormWinningHand 检查是否是通用胡牌牌型 例如 AA BCD BCD BCD BCD
func CanFormWinningHand(tileCountsOrig map[int]int, hasPair bool) bool {
	// 过滤掉数量为0的牌
	tileCounts := make(map[int]int)
	for k, v := range tileCountsOrig {
		if v > 0 {
			tileCounts[k] = v
		}
	}

	// 查找第一张牌
	if len(tileCounts) == 0 {
		return hasPair
	}

	for key, number := range tileCounts {
		// 尝试组成刻子
		if number >= 3 {
			tileCounts[key] = number - 3
			if CanFormWinningHand(tileCounts, hasPair) {
				return true
			}
			tileCounts[key] = number
		}

		// 尝试组成将
		if !hasPair && tileCounts[key] >= 2 {
			tileCounts[key] = tileCounts[key] - 2
			if CanFormWinningHand(tileCounts, true) {
				return true
			}
			tileCounts[key] = number
		}

		// 尝试组成顺子（仅对万、条、饼有效）
		if key <= 27 && key%10 <= 7 {
			if tileCounts[key] > 0 && tileCounts[key+1] > 0 && tileCounts[key+2] > 0 {
				tileCounts[key] = number - 1
				tileCounts[key+1] = tileCounts[key+1] - 1
				tileCounts[key+2] = tileCounts[key+2] - 1
				if CanFormWinningHand(tileCounts, hasPair) {
					return true
				}
				tileCounts[key] = number
				tileCounts[key+1] = tileCounts[key+1] + 1
				tileCounts[key+2] = tileCounts[key+2] + 1
			}
		}
	}
	return false
}

// sevenPair 检查是否是特殊胡牌牌型 七小对 AA BB CC DD EE FF GG
func sevenPair(seat *Seat, mahjong Mahjong) bool {
	numbers := buildNumberSlice(seat, mahjong, false)
	if len(numbers) != FullHandSize {
		return false
	}

	countMap := buildCountMap(numbers)
	for _, count := range countMap {
		// 每种牌必须是2张或4张
		if count != PairCount && count != 4 {
			return false
		}
	}
	return true
}

// loongSevenPair 检查是否是龙七对
func loongSevenPair(seat *Seat, mahjong Mahjong) bool {
	numbers := buildNumberSlice(seat, mahjong, false)
	if len(numbers) != FullHandSize {
		return false
	}

	countMap := buildCountMap(numbers)
	hasFourOfKind := false

	for _, count := range countMap {
		if count == 4 {
			hasFourOfKind = true
		} else if count != PairCount {
			// 龙七对必须全是对子，其中至少一个是四张
			return false
		}
	}
	return hasFourOfKind
}

// twoFiveEight 检查是否是特殊胡牌牌型 258 手牌或者公开牌全是 2 5 8
func twoFiveEight(seat *Seat, mahjong Mahjong) bool {
	numbers := buildNumberSlice(seat, mahjong, true)
	for _, num := range numbers {
		if !isTwoFiveEightNumber(num) {
			return false
		}
	}
	return true
}

// twoColor 检查是否是特殊胡牌牌型 上手缺一门
func twoColor(supplierType SupplierType, seat *Seat, mahjong Mahjong) bool {
	if !checkSpecial(supplierType, seat) {
		return false
	}

	numbers := buildNumberSlice(seat, mahjong, false)
	colorSet := make(map[int]bool, 3)
	for _, num := range numbers {
		colorSet[num/10] = true
	}
	return len(colorSet) < 3
}

// noJiang 检查是否是特殊胡牌牌型 上手没有 2 5 8
func noJiang(supplierType SupplierType, seat *Seat, mahjong Mahjong) bool {
	if !checkSpecial(supplierType, seat) {
		return false
	}

	numbers := buildNumberSlice(seat, mahjong, false)
	for _, num := range numbers {
		if isTwoFiveEightNumber(num) {
			return false
		}
	}
	return true
}

// checkSpecial 检查是否是第一次上手 有碰 有杠就算破坏了
func checkSpecial(supplierType SupplierType, seat *Seat) bool {
	return supplierType == CATCH &&
		seat.Step().Load() == 1 &&
		len(seat.ExtraList()) == StandardHandSize
}
