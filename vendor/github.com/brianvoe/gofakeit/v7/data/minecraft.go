package data

// Minecraft consists of various minecraft items
var Minecraft = map[string][]string{
	"ore":             {"coal", "copper", "iron", "gold", "redstone", "lapis", "diamond", "emerald"},
	"wood":            {"oak", "spruce", "birch", "jungle", "acacia", "dark oak"},
	"armortier":       {"leather", "chainmail", "iron", "gold", "diamond", "netherite"},
	"armorpart":       {"helmet", "chestplate", "leggings", "boots"},
	"weapon":          {"sword", "bow", "arrow", "trident", "shield"},
	"tool":            {"pickaxe", "axe", "shovel", "hoe", "fishing rod"},
	"dye":             {"white", "orange", "magenta", "light blue", "yellow", "lime", "pink", "gray", "light gray", "cyan", "purple", "blue", "brown", "green", "red", "black"},
	"food":            {"apple", "baked potato", "beetroot", "beetroot soup", "bread", "cake", "carrot", "chorus fruit", "cooked chicken", "cooked cod", "cooked mutton", "cooked salmon", "cookie", "enchanted golden apple", "golden apple", "glow berry", "golden carrot", "honey bottle", "melon slice", "mushroom stew", "poisonous potato", "potato", "pufferfish", "pumpkin pie", "rabbit stew", "raw beef", "raw chicken", "raw cod", "raw mutton", "raw porkchop", "raw rabbit", "raw salmon", "rotten flesh", "spider eye", "steak", "suspicous stew", "sweet berry", "tropical fish"},
	"animal":          {"chicken", "cow", "pig", "rabbit", "sheep", "wolf"},
	"villagerjob":     {"armourer", "butcher", "carpenter", "cleric", "farmer", "fisherman", "fletcher", "leatherworker", "librarian", "mason", "nitwit", "shepherd", "toolsmith", "weaponsmith"},
	"villagerstation": {"composter", "smoker", "barrel", "loom", "blast furnace", "brewing stand", "cauldron", "fletching table", "cartography table", "lectern", "smithing table", "stonecutter", "grindstone"},
	"villagerlevel":   {"novice", "apprentice", "journeyman", "expert", "master"},
	"mobpassive":      {"axolotl", "bat", "cat", "chicken", "cod", "cow", "donkey", "fox", "glow squid", "horse", "mooshroom", "mule", "ocelot", "parrot", "pig", "pufferfish", "rabbit", "salmon", "sheep", "skeleton horse", "snow golem", "squid", "strider", "tropical fish", "turtle", "villager", "wandering trader"},
	"mobneutral":      {"bee", "cave spider", "dolphin", "enderman", "goat", "iron golem", "llama", "panda", "piglin", "polar bear", "spider", "trader llama", "wolf", "zombified piglin"},
	"mobhostile":      {"blaze", "chicken jockey", "creeper", "drowned", "elder guardian", "endermite", "evoker", "ghast", "guardian", "hoglin phantom", "husk", "magma cube", "phantom", "piglin brute", "pillager", "ravager", "shulker", "silverfish", "skeleton", "skeleton horseman", "slime", "spider jockey", "stray", "vex", "vindicator", "witch", "wither skeleton", "zoglin", "zombie", "zombie villager"},
	"mobboss":         {"ender dragon", "wither"},
	"biome":           {"plain", "forest", "jungle", "mountain", "desert", "taiga", "snowy tundra", "ice spike", "swamp", "savannah", "badlands", "beach", "stone shore", "river", "ocean", "mushroom island", "the nether", "the end"},
	"weather":         {"clear", "rain", "thunder"},
}
