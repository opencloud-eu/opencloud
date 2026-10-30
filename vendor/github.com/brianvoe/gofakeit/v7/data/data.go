package data

// Data consists of the main set of fake information
var Data = map[string]map[string][]string{
	"person":    Person,
	"address":   Address,
	"company":   Company,
	"job":       Job,
	"lorem":     Lorem,
	"language":  Languages,
	"internet":  Internet,
	"file":      Files,
	"color":     Colors,
	"computer":  Computer,
	"hipster":   Hipster,
	"beer":      Beer,
	"hacker":    Hacker,
	"animal":    Animal,
	"currency":  Currency,
	"log_level": LogLevels,
	"timezone":  TimeZone,
	"car":       Car,
	"emoji":     Emoji,
	"word":      Word,
	"sentence":  Sentence,
	"food":      Food,
	"minecraft": Minecraft,
	"celebrity": Celebrity,
	"error":     Error,
	"html":      Html,
	"book":      Books,
	"movie":     Movies,
	"school":    School,
	"song":      Songs,
	"product":   Product,
	"bank":      Bank,
}

func List() map[string][]string {
	var list = make(map[string][]string)

	// Loop through the data and add the keys to the list
	for key := range Data {
		list[key] = []string{}

		// Loop through the sub data and add the keys to the list
		for subkey := range Data[key] {
			list[key] = append(list[key], subkey)
		}
	}

	return list
}

func Get(key string) map[string][]string {
	// Make sure the key exists, if not return an empty map
	if _, ok := Data[key]; !ok {
		return make(map[string][]string)
	}

	return Data[key]
}

func Set(key string, data map[string][]string) {
	Data[key] = data
}

func Remove(key string) {
	delete(Data, key)
}

func GetSubData(key, subkey string) []string {
	// Make sure the key exists, if not return an empty map
	if _, ok := Data[key]; !ok {
		return []string{}
	}

	return Data[key][subkey]
}

func SetSub(key, subkey string, data []string) {
	// Make sure the key exists, if not add it
	if _, ok := Data[key]; !ok {
		Data[key] = make(map[string][]string)
	}

	Data[key][subkey] = data
}

func RemoveSub(key, subkey string) {
	delete(Data[key], subkey)
}
