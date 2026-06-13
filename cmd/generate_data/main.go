package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"time"
)

type Category struct {
	L1 string
	L2 []string
	L3 map[string][]string
}

var categories = []Category{
	{
		L1: "餐饮",
		L2: []string{"中餐", "西餐", "日料", "咖啡", "快餐", "火锅", "烧烤", "甜点"},
		L3: map[string][]string{
			"中餐": {"川菜", "粤菜", "鲁菜", "苏菜", "浙菜", "湘菜", "徽菜", "闽菜"},
			"西餐": {"牛排", "意大利菜", "法国菜", "墨西哥菜"},
			"日料": {"寿司", "拉面", "居酒屋", "天妇罗"},
			"咖啡": {"连锁咖啡", "精品咖啡", "咖啡培训"},
			"快餐": {"汉堡", "炸鸡", "中式快餐"},
			"火锅": {"四川火锅", "潮汕火锅", "铜锅涮肉"},
			"烧烤": {"韩式烧烤", "中式烧烤", "日式烧肉"},
			"甜点": {"蛋糕", "冰淇淋", "面包店", "甜品店"},
		},
	},
	{
		L1: "购物",
		L2: []string{"购物中心", "超市", "便利店", "专卖店", "市场"},
		L3: map[string][]string{
			"购物中心": {"综合商场", "奥特莱斯", "奢侈品中心"},
			"超市":     {"大型超市", "精品超市", "生鲜超市"},
			"便利店":   {"24小时便利店", "社区便利店"},
			"专卖店":   {"服装", "电子产品", "化妆品", "运动户外"},
			"市场":     {"农贸市场", "小商品市场", "建材市场"},
		},
	},
	{
		L1: "酒店",
		L2: []string{"经济型酒店", "中端酒店", "高端酒店", "民宿"},
		L3: map[string][]string{
			"经济型酒店": {"连锁快捷", "青年旅舍"},
			"中端酒店":   {"商务酒店", "精品酒店"},
			"高端酒店":   {"五星级酒店", "度假村"},
			"民宿":     {"城市民宿", "乡村民宿"},
		},
	},
	{
		L1: "景点",
		L2: []string{"历史古迹", "自然风光", "主题公园", "博物馆"},
		L3: map[string][]string{
			"历史古迹": {"宫殿", "寺庙", "园林", "遗址"},
			"自然风光": {"山岳", "湖泊", "公园", "海滩"},
			"主题公园": {"游乐园", "动物园", "植物园", "水族馆"},
			"博物馆":   {"综合博物馆", "专题博物馆", "美术馆"},
		},
	},
	{
		L1: "交通",
		L2: []string{"地铁站", "公交站", "火车站", "机场", "停车场"},
		L3: map[string][]string{
			"地铁站":   {"换乘站", "普通站"},
			"公交站":   {"枢纽站", "普通站"},
			"火车站":   {"高铁站", "普通站"},
			"机场":     {"国际机场", "国内机场"},
			"停车场":   {"公共停车场", "商业停车场"},
		},
	},
	{
		L1: "医疗",
		L2: []string{"医院", "诊所", "药店", "体检中心"},
		L3: map[string][]string{
			"医院":    {"综合医院", "专科医院", "中医院"},
			"诊所":    {"社区诊所", "私人诊所"},
			"药店":    {"连锁药店", "24小时药店"},
			"体检中心": {"专业体检", "医院体检中心"},
		},
	},
	{
		L1: "教育",
		L2: []string{"学校", "培训机构", "图书馆", "幼儿园"},
		L3: map[string][]string{
			"学校":   {"小学", "中学", "大学"},
			"培训机构": {"语言培训", "IT培训", "艺术培训", "课外辅导"},
			"图书馆":  {"公共图书馆", "大学图书馆"},
			"幼儿园":  {"公立幼儿园", "私立幼儿园"},
		},
	},
	{
		L1: "休闲",
		L2: []string{"电影院", "KTV", "健身房", "棋牌室", "酒吧"},
		L3: map[string][]string{
			"电影院": {"IMAX影院", "普通影院", "私人影院"},
			"KTV":   {"量贩式KTV", "商务KTV"},
			"健身房": {"连锁健身", "精品健身", "瑜伽馆"},
			"棋牌室": {"麻将馆", "桌游吧"},
			"酒吧":   {"清吧", "夜店", "鸡尾酒吧"},
		},
	},
	{
		L1: "生活服务",
		L2: []string{"银行", "邮局", "理发店", "洗衣店", "维修店"},
		L3: map[string][]string{
			"银行":   {"工商银行", "建设银行", "农业银行", "中国银行"},
			"邮局":   {"邮政支局", "快递网点"},
			"理发店": {"美发店", "美容美发", "男士理发"},
			"洗衣店": {"干洗店", "洗衣工厂"},
			"维修店": {"家电维修", "手机维修", "汽车维修", "咖啡机维修"},
		},
	},
	{
		L1: "政府机构",
		L2: []string{"派出所", "街道办", "税务局", "工商局"},
		L3: map[string][]string{
			"派出所":   {"户籍派出所", "治安派出所"},
			"街道办":   {"街道办事处", "社区居委会"},
			"税务局":   {"国税", "地税"},
			"工商局":   {"市场监管局", "工商所"},
		},
	},
}

type POIName struct {
	Zh  string
	En  string
	Ja  string
	Ko  string
	L1  string
	L2  string
	L3  string
}

var poiTemplates = []POIName{
	{"星巴克咖啡", "Starbucks Coffee", "スターバックス コーヒー", "스타벅스 커피", "餐饮", "咖啡", "连锁咖啡"},
	{"星巴克臻选", "STARBUCKS RESERVE", "スターバックス リザーブ", "스타벅스 리저브", "餐饮", "咖啡", "精品咖啡"},
	{"瑞幸咖啡", "Luckin Coffee", "ルッキンコーヒー", "럭킨커피", "餐饮", "咖啡", "连锁咖啡"},
	{"麦当劳", "McDonald's", "マクドナルド", "맥도날드", "餐饮", "快餐", "汉堡"},
	{"肯德基", "KFC", "ケンタッキー", "KFC", "餐饮", "快餐", "炸鸡"},
	{"海底捞火锅", "Haidilao Hot Pot", "ハイディラオ火鍋", "하이디라오 훠궈", "餐饮", "火锅", "四川火锅"},
	{"喜茶", "HeyTea", "ヘイティー", "헤이티", "餐饮", "甜点", "甜品店"},
	{"奈雪的茶", "Nayuki", "ナユキ", "나유키", "餐饮", "甜点", "甜品店"},
	{"优衣库", "UNIQLO", "ユニクロ", "유니클로", "购物", "专卖店", "服装"},
	{"无印良品", "MUJI", "無印良品", "무인양품", "购物", "专卖店", "服装"},
	{"宜家家居", "IKEA", "イケア", "이케아", "购物", "专卖店", "家居"},
	{"王府井百货", "Wangfujing Department Store", "王府井百貨", "왕푸징 백화점", "购物", "购物中心", "综合商场"},
	{"北京apm", "Beijing apm", "北京apm", "베이징 apm", "购物", "购物中心", "综合商场"},
	{"东方新天地", "Oriental Plaza", "オリエンタルプラザ", "동방신천지", "购物", "购物中心", "综合商场"},
	{"北京SKP", "Beijing SKP", "北京SKP", "베이징 SKP", "购物", "购物中心", "奢侈品中心"},
	{"国贸商城", "China World Mall", "チャイナワールドモール", "국제무역상성", "购物", "购物中心", "综合商场"},
	{"三里屯太古里", "Taikoo Li Sanlitun", "三里屯タイクーリ", "싼리툰 타이다리", "购物", "购物中心", "综合商场"},
	{"7-Eleven便利店", "7-Eleven", "セブンイレブン", "세븐일레븐", "购物", "便利店", "24小时便利店"},
	{"全家便利店", "FamilyMart", "ファミリーマート", "패밀리마트", "购物", "便利店", "24小时便利店"},
	{"罗森便利店", "Lawson", "ローソン", "로손", "购物", "便利店", "24小时便利店"},
	{"北京饭店", "Beijing Hotel", "北京飯店", "베이징 호텔", "酒店", "高端酒店", "五星级酒店"},
	{"王府井希尔顿酒店", "Hilton Beijing Wangfujing", "ヒルトン北京王府井", "힐튼 베이징 왕푸징", "酒店", "高端酒店", "五星级酒店"},
	{"如家酒店", "Home Inn", "ホームイン", "호텔", "酒店", "经济型酒店", "连锁快捷"},
	{"汉庭酒店", "Hanting Hotel", "ハンティンホテル", "한팅 호텔", "酒店", "经济型酒店", "连锁快捷"},
	{"全季酒店", "JI Hotel", "ジーホテル", "지 호텔", "酒店", "中端酒店", "商务酒店"},
	{"故宫博物院", "The Palace Museum", "故宮博物院", "고궁 박물관", "景点", "历史古迹", "宫殿"},
	{"天安门广场", "Tian'anmen Square", "天安門広場", "천안문 광장", "景点", "历史古迹", "广场"},
	{"王府井步行街", "Wangfujing Pedestrian Street", "王府井歩行街", "왕푸징 보행거리", "景点", "历史古迹", "商业街"},
	{"北海公园", "Beihai Park", "北海公園", "북해공원", "景点", "自然风光", "公园"},
	{"景山公园", "Jingshan Park", "景山公園", "경산공원", "景点", "自然风光", "公园"},
	{"中国国家博物馆", "National Museum of China", "中国国家博物館", "중국국가박물관", "景点", "博物馆", "综合博物馆"},
	{"北京动物园", "Beijing Zoo", "北京動物園", "베이징 동물원", "景点", "主题公园", "动物园"},
	{"北京欢乐谷", "Happy Valley Beijing", "北京ハッピーバレー", "베이징 해피밸리", "景点", "主题公园", "游乐园"},
	{"西单地铁站", "Xidan Metro Station", "西单地下鉄駅", "시단 지하철역", "交通", "地铁站", "换乘站"},
	{"王府井地铁站", "Wangfujing Metro Station", "王府井地下鉄駅", "왕푸징 지하철역", "交通", "地铁站", "普通站"},
	{"东单地铁站", "Dongdan Metro Station", "東単地下鉄駅", "둥단 지하철역", "交通", "地铁站", "换乘站"},
	{"北京火车站", "Beijing Railway Station", "北京駅", "베이징 기차역", "交通", "火车站", "普通站"},
	{"北京南站", "Beijing South Railway Station", "北京南駅", "베이징난 기차역", "交通", "火车站", "高铁站"},
	{"首都国际机场", "Beijing Capital International Airport", "北京首都国際空港", "베이징 수도 국제공항", "交通", "机场", "国际机场"},
	{"北京协和医院", "Peking Union Medical College Hospital", "北京協和医院", "베이징 혜화병원", "医疗", "医院", "综合医院"},
	{"北京医院", "Beijing Hospital", "北京病院", "베이징 병원", "医疗", "医院", "综合医院"},
	{"同仁堂药店", "Tongrentang Pharmacy", "同仁堂薬店", "동인당 약국", "医疗", "药店", "连锁药店"},
	{"北京大学", "Peking University", "北京大学", "북경대학", "教育", "学校", "大学"},
	{"清华大学", "Tsinghua University", "清華大学", "칭화대학", "教育", "学校", "大学"},
	{"北京景山学校", "Beijing Jingshan School", "北京景山学校", "베이징 징산 학교", "教育", "学校", "中学"},
	{"国家图书馆", "National Library of China", "中国国家図書館", "중국국가도서관", "教育", "图书馆", "公共图书馆"},
	{"首都电影院", "Capital Cinema", "首都映画館", "수도 영화관", "休闲", "电影院", "普通影院"},
	{"万达影城", "Wanda Cinema", "ワンダシネマ", "완다 시네마", "休闲", "电影院", "IMAX影院"},
	{"钱柜KTV", "Cashbox KTV", "キャッシュボックスKTV", "캐쉬박스 KTV", "休闲", "KTV", "量贩式KTV"},
	{"中体倍力健身", "Zhongti Beili Fitness", "中体倍力フィットネス", "중티베이리 피트니스", "休闲", "健身房", "连锁健身"},
	{"工体酒吧街", "Worker's Stadium Bar Street", "工人体育場バー街", "노동자체육장 바 거리", "休闲", "酒吧", "夜店"},
	{"MIX酒吧", "MIX Club", "MIXクラブ", "MIX 클럽", "休闲", "酒吧", "鸡尾酒吧"},
	{"中国银行", "Bank of China", "中国銀行", "중국은행", "生活服务", "银行", "中国银行"},
	{"工商银行", "ICBC", "工商銀行", "공상은행", "生活服务", "银行", "工商银行"},
	{"建设银行", "China Construction Bank", "建設銀行", "건설은행", "生活服务", "银行", "建设银行"},
	{"顺丰速运", "SF Express", "SF速達", "순익스프레스", "生活服务", "邮局", "快递网点"},
	{"EMS邮政", "EMS", "EMS", "EMS", "生活服务", "邮局", "邮政支局"},
	{"文峰美发", "Wenfeng Hair Salon", "文峰美容", "문장 미용실", "生活服务", "理发店", "美发店"},
	{"永琪美容美发", "Yongqi Beauty Salon", "永琪美容美容", "영기 미용실", "生活服务", "理发店", "美容美发"},
	{"福奈特洗衣", "Fornet Laundry", "フォルネットランドリー", "포르넷 세탁소", "生活服务", "洗衣店", "干洗店"},
	{"苹果手机维修", "Apple Repair", "アップル修理", "애플 수리", "生活服务", "维修店", "手机维修"},
	{"星巴克咖啡培训中心", "Starbucks Coffee Training", "スターバックスコーヒートレーニング", "스타벅스 커피 교육", "餐饮", "咖啡", "咖啡培训"},
	{"咖啡机维修服务", "Coffee Machine Repair", "コーヒーマシン修理", "커피머신 수리", "生活服务", "维修店", "咖啡机维修"},
	{"东华门派出所", "Donghuamen Police Station", "東華門派出所", "둥화먼 파출소", "政府机构", "派出所", "户籍派出所"},
	{"王府井街道办", "Wangfujing Sub-district Office", "王府井街道弁事処", "왕푸징 가도판공실", "政府机构", "街道办", "街道办事处"},
	{"东城区税务局", "Dongcheng District Tax Bureau", "東城区税務局", "둥청구 세무국", "政府机构", "税务局", "国税"},
	{"东城区工商局", "Dongcheng District Market Supervision Bureau", "東城区市場監督管理局", "둥청구 시장감독국", "政府机构", "工商局", "市场监管局"},
}

var districts = []string{
	"东城区", "西城区", "朝阳区", "海淀区", "丰台区", "石景山区",
}

var streets = map[string][]string{
	"东城区":  {"王府井大街", "东长安街", "东单北大街", "建国门内大街", "朝阳门南小街", "东直门内大街", "和平里西街", "安定门东大街"},
	"西城区":  {"西单北大街", "西长安街", "宣武门西大街", "复兴门内大街", "阜成门内大街", "西直门外大街", "德胜门东大街"},
	"朝阳区":  {"三里屯路", "建国路", "朝阳门外大街", "工体北路", "光华路", "东三环中路", "建国门外大街", "朝阳北路"},
	"海淀区":  {"中关村大街", "海淀南路", "学院路", "清华西路", "颐和园路", "西直门外大街", "北四环西路"},
	"丰台区":  {"丰台路", "南三环西路", "丽泽路", "丰台北路", "马家堡西路", "方庄路"},
	"石景山区": {"石景山路", "古城大街", "鲁谷路", "八角西街", "杨庄大街"},
}

type Hotspot struct {
	Name      string
	Lat       float64
	Lng       float64
	Weight    int
}

var beijingHotspots = []Hotspot{
	{"王府井", 39.9147, 116.4108, 80},
	{"东单", 39.9145, 116.4200, 60},
	{"西单", 39.9135, 116.3743, 50},
	{"天安门", 39.9055, 116.3976, 70},
	{"三里屯", 39.9370, 116.4520, 55},
	{"国贸CBD", 39.9087, 116.4605, 50},
	{"中关村", 39.9831, 116.3160, 45},
	{"西二旗", 40.0499, 116.3004, 40},
	{"望京", 39.9966, 116.4788, 45},
	{"五道口", 39.9929, 116.3466, 35},
	{"东直门", 39.9415, 116.4341, 40},
	{"西直门", 39.9409, 116.3545, 35},
	{"朝阳门", 39.9258, 116.4352, 35},
	{"建国门", 39.9088, 116.4356, 35},
	{"崇文门", 39.8996, 116.4232, 30},
	{"宣武门", 39.8992, 116.3705, 30},
	{"复兴门", 39.9071, 116.3595, 30},
	{"积水潭", 39.9492, 116.3743, 25},
	{"鼓楼", 39.9406, 116.3950, 30},
	{"南锣鼓巷", 39.9376, 116.4036, 35},
}

func main() {
	rand.Seed(time.Now().UnixNano())

	os.MkdirAll("data", 0755)

	file, err := os.Create("data/beijing_test_data.csv")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"name_zh", "name_en", "name_ja", "name_ko",
		"category_l1", "category_l2", "category_l3",
		"address", "city", "district",
		"lat", "lng",
		"phone", "business_hours",
		"rating", "tags",
		"status", "source",
	}
	writer.Write(headers)

	totalPOIs := 1000
	generated := 0

	for generated < totalPOIs {
		hotspot := pickHotspot()
		template := poiTemplates[rand.Intn(len(poiTemplates))]

		latOffset := (rand.Float64() - 0.5) * 0.04
		lngOffset := (rand.Float64() - 0.5) * 0.05

		lat := hotspot.Lat + latOffset
		lng := hotspot.Lng + lngOffset

		district := pickDistrict()
		streetList := streets[district]
		street := streetList[rand.Intn(len(streetList))]
		houseNum := rand.Intn(200) + 1
		address := fmt.Sprintf("%s%d号", street, houseNum)

		phone := generatePhone()
		businessHours := generateBusinessHours(template.L2)
		rating := 3.0 + rand.Float64()*2.0
		rating = float64(int(rating*10)) / 10.0

		tags := generateTags(template)

		branchName := ""
		if rand.Float32() < 0.4 {
			branchName = fmt.Sprintf("(%s店)", hotspot.Name)
		}

		sources := []string{"manual", "crawler", "partner"}
		source := sources[rand.Intn(len(sources))]

		record := []string{
			template.Zh + branchName,
			template.En + branchName,
			template.Ja + branchName,
			template.Ko + branchName,
			template.L1,
			template.L2,
			template.L3,
			address,
			"北京",
			district,
			fmt.Sprintf("%.6f", lat),
			fmt.Sprintf("%.6f", lng),
			phone,
			businessHours,
			fmt.Sprintf("%.1f", rating),
			tags,
			"active",
			source,
		}

		writer.Write(record)
		generated++
	}

	fmt.Printf("Generated %d POIs to data/beijing_test_data.csv\n", totalPOIs)
}

func pickHotspot() Hotspot {
	totalWeight := 0
	for _, h := range beijingHotspots {
		totalWeight += h.Weight
	}

	r := rand.Intn(totalWeight)
	for _, h := range beijingHotspots {
		if r < h.Weight {
			return h
		}
		r -= h.Weight
	}
	return beijingHotspots[0]
}

func pickDistrict() string {
	weights := []int{25, 20, 25, 20, 7, 3}
	total := 0
	for _, w := range weights {
		total += w
	}

	r := rand.Intn(total)
	for i, w := range weights {
		if r < w {
			return districts[i]
		}
		r -= w
	}
	return districts[0]
}

func generatePhone() string {
	areaCodes := []string{"010-5", "010-6", "010-8"}
	areaCode := areaCodes[rand.Intn(len(areaCodes))]
	number := rand.Intn(90000000) + 10000000
	return fmt.Sprintf("%s%d", areaCode, number)
}

func generateBusinessHours(categoryL2 string) string {
	switch categoryL2 {
	case "便利店":
		return "周一:00:00-24:00;周二:00:00-24:00;周三:00:00-24:00;周四:00:00-24:00;周五:00:00-24:00;周六:00:00-24:00;周日:00:00-24:00"
	case "酒吧", "夜店":
		return "周一:18:00-02:00;周二:18:00-02:00;周三:18:00-02:00;周四:18:00-02:00;周五:18:00-03:00;周六:18:00-03:00;周日:18:00-01:00"
	case "咖啡":
		return "周一:07:00-22:00;周二:07:00-22:00;周三:07:00-22:00;周四:07:00-22:00;周五:07:00-23:00;周六:08:00-23:00;周日:08:00-22:00"
	case "快餐", "甜点":
		return "周一:09:00-22:00;周二:09:00-22:00;周三:09:00-22:00;周四:09:00-22:00;周五:09:00-23:00;周六:09:00-23:00;周日:09:00-22:00"
	case "酒店", "医院", "药店":
		return "周一:00:00-24:00;周二:00:00-24:00;周三:00:00-24:00;周四:00:00-24:00;周五:00:00-24:00;周六:00:00-24:00;周日:00:00-24:00"
	default:
		return "周一:09:00-21:00;周二:09:00-21:00;周三:09:00-21:00;周四:09:00-21:00;周五:09:00-22:00;周六:10:00-22:00;周日:10:00-21:00"
	}
}

func generateTags(template POIName) string {
	baseTags := []string{template.L2}

	switch template.L2 {
	case "咖啡":
		baseTags = append(baseTags, "wifi", "可堂食", "可外带")
		if template.L3 == "精品咖啡" {
			baseTags = append(baseTags, "精品")
		}
	case "中餐":
		baseTags = append(baseTags, "可堂食", "可外带", "有包间")
	case "火锅":
		baseTags = append(baseTags, "可堂食", "有包间", "聚餐")
	case "快餐":
		baseTags = append(baseTags, "快餐", "可外带")
	case "购物中心":
		baseTags = append(baseTags, "停车场", "wifi", "空调")
	case "酒店":
		baseTags = append(baseTags, "停车场", "wifi", "早餐")
	}

	if rand.Float32() < 0.3 {
		baseTags = append(baseTags, "推荐")
	}
	if rand.Float32() < 0.2 {
		baseTags = append(baseTags, "新店")
	}
	if rand.Float32() < 0.15 {
		baseTags = append(baseTags, "老字号")
	}

	return joinStrings(baseTags, "|")
}

func joinStrings(s []string, sep string) string {
	result := ""
	for i, str := range s {
		if i > 0 {
			result += sep
		}
		result += str
	}
	return result
}
