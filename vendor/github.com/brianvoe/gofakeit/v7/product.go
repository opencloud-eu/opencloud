package gofakeit

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/brianvoe/gofakeit/v7/data"
)

type ProductInfo struct {
	Name        string   `json:"name" xml:"name"`
	Description string   `json:"description" xml:"description"`
	Categories  []string `json:"categories" xml:"categories"`
	Price       float64  `json:"price" xml:"price"`
	Features    []string `json:"features" xml:"features"`
	Color       string   `json:"color" xml:"color"`
	Material    string   `json:"material" xml:"material"`
	UPC         string   `json:"upc" xml:"upc"`
	Audience    []string `json:"audience" xml:"audience"`
	Dimension   string   `json:"dimension" xml:"dimension"`
	UseCase     string   `json:"use_case" xml:"use_case"`
	Benefit     string   `json:"benefit" xml:"benefit"`
	Suffix      string   `json:"suffix" xml:"suffix"`
}

// Product will generate a random set of product information
func Product() *ProductInfo { return product(GlobalFaker) }

// Product will generate a random set of product information
func (f *Faker) Product() *ProductInfo { return product(f) }

func product(f *Faker) *ProductInfo {
	// Categories
	categories := []string{}
	weightedCategory, _ := weighted(f, []any{1, 2, 3, 4}, []float32{1, 4, 3, 4})

	for i := 0; i < weightedCategory.(int); i++ {
		categories = append(categories, productCategory(f))
	}

	// Features
	features := []string{}
	for i := 0; i < number(f, 1, 5); i++ {
		features = append(features, productFeature(f))
	}

	product := &ProductInfo{
		Name:        productName(f),
		Description: productDescription(f),
		Categories:  categories,
		Price:       price(f, 3.00, 100.00),
		UPC:         productUPC(f),
		Features:    features,
		Color:       safeColor(f),
		Material:    productMaterial(f),
		Audience:    productAudience(f),
		Dimension:   productDimension(f),
		UseCase:     productUseCase(f),
		Benefit:     productBenefit(f),
		Suffix:      productSuffix(f),
	}

	return product
}

// ProductName will generate a random product name
func ProductName() string { return productName(GlobalFaker) }

// ProductName will generate a random product name
func (f *Faker) ProductName() string { return productName(f) }

func productName(f *Faker) string {
	name := getRandValue(f, []string{"product", "name"})
	switch number(f, 0, 9) {
	case 1:
		// Name + Adjective + Feature
		return title(fmt.Sprintf("%s %s %s", name, getRandValue(f, []string{"product", "adjective"}), productFeature(f)))
	case 2:
		// Adjective + Material + Name
		return title(fmt.Sprintf("%s %s %s", getRandValue(f, []string{"product", "adjective"}), productMaterial(f), name))
	case 3:
		// Color + Name + Suffix
		return title(fmt.Sprintf("%s %s %s", safeColor(f), name, getRandValue(f, []string{"product", "suffix"})))
	case 4:
		// Feature + Name + Adjective
		return title(fmt.Sprintf("%s %s %s", productFeature(f), name, getRandValue(f, []string{"product", "adjective"})))
	case 5:
		// Material + Color + Name
		return title(fmt.Sprintf("%s %s %s", productMaterial(f), safeColor(f), name))
	case 6:
		// Name + Suffix + Material
		return title(fmt.Sprintf("%s %s %s", name, getRandValue(f, []string{"product", "suffix"}), productMaterial(f)))
	case 7:
		// Adjective + Feature + Name
		return title(fmt.Sprintf("%s %s %s", getRandValue(f, []string{"product", "adjective"}), productFeature(f), name))
	case 8:
		// Color + Material + Name
		return title(fmt.Sprintf("%s %s %s", safeColor(f), productMaterial(f), name))
	case 9:
		// Suffix + Adjective + Name
		return title(fmt.Sprintf("%s %s %s", getRandValue(f, []string{"product", "suffix"}), getRandValue(f, []string{"product", "adjective"}), name))
	}

	// case: 0 - Adjective + Name + Suffix
	return title(fmt.Sprintf("%s %s %s", getRandValue(f, []string{"product", "adjective"}), name, getRandValue(f, []string{"product", "suffix"})))
}

// ProductDescription will generate a random product description
func ProductDescription() string { return productDescription(GlobalFaker) }

// ProductDescription will generate a random product description
func (f *Faker) ProductDescription() string { return productDescription(f) }

func productDescription(f *Faker) string {
	prodDesc := getRandValue(f, []string{"product", "description"})

	// Replace all {productaudience} with join "and"
	for strings.Contains(prodDesc, "{productaudience}") {
		prodDesc = strings.Replace(prodDesc, "{productaudience}", strings.Join(productAudience(f), " and "), 1)
	}

	desc, _ := generate(f, prodDesc)
	return desc
}

// ProductCategory will generate a random product category
func ProductCategory() string { return productCategory(GlobalFaker) }

// ProductCategory will generate a random product category
func (f *Faker) ProductCategory() string { return productCategory(f) }

func productCategory(f *Faker) string {
	return getRandValue(f, []string{"product", "category"})
}

// ProductFeature will generate a random product feature
func ProductFeature() string { return productFeature(GlobalFaker) }

// ProductFeature will generate a random product feature
func (f *Faker) ProductFeature() string { return productFeature(f) }

func productFeature(f *Faker) string {
	return getRandValue(f, []string{"product", "feature"})
}

// ProductMaterial will generate a random product material
func ProductMaterial() string { return productMaterial(GlobalFaker) }

// ProductMaterial will generate a random product material
func (f *Faker) ProductMaterial() string { return productMaterial(f) }

func productMaterial(f *Faker) string {
	return getRandValue(f, []string{"product", "material"})
}

// ProductUPC will generate a random product UPC
func ProductUPC() string { return productUPC(GlobalFaker) }

// ProductUPC will generate a random product UPC
func (f *Faker) ProductUPC() string { return productUPC(f) }

func productUPC(f *Faker) string {
	// The first digit of a UPC is a fixed digit (usually 0)
	upc := "0"

	// Generate the remaining 11 digits randomly
	for i := 1; i < 12; i++ {
		digit := number(f, 0, 9)
		upc += fmt.Sprintf("%d", digit)
	}

	return upc
}

// ProductAudience will generate a random target audience
func ProductAudience() []string { return productAudience(GlobalFaker) }

// ProductAudience will generate a random target audience
func (f *Faker) ProductAudience() []string { return productAudience(f) }

func productAudience(f *Faker) []string {
	audiences := []string{}
	for i := 0; i < number(f, 1, 2); i++ {
		// Check if the target audience is already in the list
		// If it is, generate a new target audience
		for {
			audience := getRandValue(f, []string{"product", "target_audience"})
			// Check if in array
			if !stringInSlice(audience, audiences) {
				audiences = append(audiences, audience)
				break
			}
		}
	}
	return audiences
}

// ProductDimension will generate a random product dimension
func ProductDimension() string { return productDimension(GlobalFaker) }

// ProductDimension will generate a random product dimension
func (f *Faker) ProductDimension() string { return productDimension(f) }

func productDimension(f *Faker) string {
	return getRandValue(f, []string{"product", "dimension"})
}

// ProductUseCase will generate a random product use case
func ProductUseCase() string { return productUseCase(GlobalFaker) }

// ProductUseCase will generate a random product use case
func (f *Faker) ProductUseCase() string { return productUseCase(f) }

func productUseCase(f *Faker) string {
	return getRandValue(f, []string{"product", "use_case"})
}

// ProductBenefit will generate a random product benefit
func ProductBenefit() string { return productBenefit(GlobalFaker) }

// ProductBenefit will generate a random product benefit
func (f *Faker) ProductBenefit() string { return productBenefit(f) }

func productBenefit(f *Faker) string {
	return getRandValue(f, []string{"product", "benefit"})
}

// ProductSuffix will generate a random product suffix
func ProductSuffix() string { return productSuffix(GlobalFaker) }

// ProductSuffix will generate a random product suffix
func (f *Faker) ProductSuffix() string { return productSuffix(f) }

func productSuffix(f *Faker) string {
	return getRandValue(f, []string{"product", "suffix"})
}

// ProductISBN13 will generate a random ISBN-13 string for the product
func ProductISBN(opts *ISBNOptions) string { return productISBN(GlobalFaker, opts) }

// ProductISBN13 will generate a random ISBN-13 string for the product
func (f *Faker) ProductISBN(opts *ISBNOptions) string { return productISBN(f, opts) }

type ISBNOptions struct {
	Version   string // "10" or "13"
	Separator string // e.g. "-", "" (default: "-")
}

func productISBN(f *Faker, opts *ISBNOptions) string {
	if opts == nil {
		opts = &ISBNOptions{Version: "13", Separator: "-"}
	}

	sep := opts.Separator
	if sep == "" {
		sep = "-"
	}

	// string of n random digits
	randomDigits := func(f *Faker, n int) string {
		digits := make([]byte, n)
		for i := 0; i < n; i++ {
			digits[i] = byte('0' + number(f, 0, 9))
		}
		return string(digits)
	}

	switch opts.Version {
	case "10":
		// ISBN-10 format: group(1)-registrant(4)-publication(3)-check(1)
		group := randomDigits(f, 1)
		registrant := randomDigits(f, 4)
		publication := randomDigits(f, 3)
		base := group + registrant + publication

		// checksum
		sum := 0
		for i, c := range base {
			digit := int(c - '0')
			sum += digit * (10 - i)
		}
		remainder := (11 - (sum % 11)) % 11
		check := "X"
		if remainder < 10 {
			check = strconv.Itoa(remainder)
		}

		return strings.Join([]string{group, registrant, publication, check}, sep)

	case "13":
		// ISBN-13 format: prefix(3)-group(1)-registrant(4)-publication(4)-check(1)
		prefix := data.ISBN13Prefix
		group := randomDigits(f, 1)
		registrant := randomDigits(f, 4)
		publication := randomDigits(f, 4)
		base := prefix + group + registrant + publication

		// checksum
		sum := 0
		for i, c := range base {
			digit := int(c - '0')
			if i%2 == 0 {
				sum += digit
			} else {
				sum += digit * 3
			}
		}
		remainder := (10 - (sum % 10)) % 10
		check := strconv.Itoa(remainder)

		return strings.Join([]string{prefix, group, registrant, publication, check}, sep)

	default:
		// fallback to ISBN-13 if invalid version provided
		return productISBN(f, &ISBNOptions{Version: "13", Separator: sep})
	}
}

func addProductLookup() {
	AddFuncLookup("product", Info{
		Display:     "Product",
		Category:    "product",
		Description: "An item created for sale or use",
		Example: `{
	"name": "olive copper monitor",
	"description": "Backwards caused quarterly without week it hungry thing someone him regularly. Whomever this revolt hence from his timing as quantity us these yours.",
	"categories": [
		"clothing",
		"tools and hardware"
	],
	"price": 7.61,
	"features": [
		"ultra-lightweight"
	],
	"color": "navy",
	"material": "brass",
	"upc": "012780949980",
	"audience": [
		"adults"
	],
	"dimension": "medium",
	"use_case": "home",
	"benefit": "comfort",
	"suffix": "pro"
}`,
		Output:      "map[string]any",
		ContentType: "application/json",
		Aliases: []string{
			"goods",
			"merchandise",
			"retail item",
			"consumer product",
			"commercial item",
		},
		Keywords: []string{
			"product", "sale", "use", "trade", "manufactured",
			"market", "inventory", "supply", "distribution", "commodity",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return product(f), nil
		},
	})

	AddFuncLookup("productname", Info{
		Display:     "Product Name",
		Category:    "product",
		Description: "Distinctive title or label assigned to a product for identification and marketing",
		Example:     "olive copper monitor",
		Output:      "string",
		Aliases: []string{
			"product title",
			"product label",
			"brand name",
			"item name",
			"product identifier",
		},
		Keywords: []string{
			"product", "name", "title", "label", "brand",
			"item", "merchandise", "goods", "article",
			"identifier", "marketing", "branding",
			"catalog", "inventory",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productName(f), nil
		},
	})

	AddFuncLookup("productdescription", Info{
		Display:     "Product Description",
		Category:    "product",
		Description: "Explanation detailing the features and characteristics of a product",
		Example:     "Backwards caused quarterly without week it hungry thing someone him regularly. Whomever this revolt hence from his timing as quantity us these yours.",
		Output:      "string",
		Aliases: []string{
			"product details",
			"product specs",
			"item description",
			"feature list",
			"marketing copy",
		},
		Keywords: []string{
			"product", "description", "details", "features",
			"specifications", "characteristics", "summary",
			"overview", "attributes", "benefits",
			"marketing", "content", "copy", "info", "text",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productDescription(f), nil
		},
	})

	AddFuncLookup("productcategory", Info{
		Display:     "Product Category",
		Category:    "product",
		Description: "Classification grouping similar products based on shared characteristics or functions",
		Example:     "clothing",
		Output:      "string",
		Aliases: []string{
			"product classification",
			"product type",
			"item category",
			"product group",
			"product segment",
		},
		Keywords: []string{
			"product", "category", "type", "class", "classification",
			"group", "segment", "line", "collection", "range",
			"electronics", "furniture", "clothing", "appliances",
			"food", "toys", "accessories", "goods",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productCategory(f), nil
		},
	})

	AddFuncLookup("productfeature", Info{
		Display:     "Product Feature",
		Category:    "product",
		Description: "Specific characteristic of a product that distinguishes it from others products",
		Example:     "ultra-lightweight",
		Output:      "string",
		Aliases: []string{
			"product trait",
			"product attribute",
			"key feature",
			"unique feature",
			"special characteristic",
		},
		Keywords: []string{
			"feature", "trait", "attribute", "characteristic",
			"capability", "functionality", "specification",
			"benefit", "advantage", "highlight",
			"unique", "differentiator",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productFeature(f), nil
		},
	})

	AddFuncLookup("productmaterial", Info{
		Display:     "Product Material",
		Category:    "product",
		Description: "The substance from which a product is made, influencing its appearance, durability, and properties",
		Example:     "brass",
		Output:      "string",
		Aliases: []string{
			"material type",
			"product substance",
			"product composition",
			"item material",
			"build material",
		},
		Keywords: []string{
			"material", "substance", "composition", "make",
			"fabric", "textile", "cloth", "leather", "wool",
			"wood", "metal", "plastic", "glass", "stone",
			"durability", "properties", "construction",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productMaterial(f), nil
		},
	})

	AddFuncLookup("productupc", Info{
		Display:     "Product UPC",
		Category:    "product",
		Description: "Standardized barcode used for product identification and tracking in retail and commerce",
		Example:     "012780949980",
		Output:      "string",
		Aliases: []string{
			"upc code",
			"product barcode",
			"product code",
			"product sku",
			"universal product code",
			"retail barcode",
		},
		Keywords: []string{
			"upc", "barcode", "product", "code", "identifier",
			"sku", "retail", "commerce", "inventory",
			"tracking", "scanning", "checkout", "label",
			"universal", "standard",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productUPC(f), nil
		},
	})

	AddFuncLookup("productaudience", Info{
		Display:     "Product Audience",
		Category:    "product",
		Description: "The group of people for whom the product is designed or intended",
		Example:     "adults",
		Output:      "[]string",
		Aliases: []string{
			"target audience",
			"target market",
			"customer group",
			"user base",
			"demographic group",
		},
		Keywords: []string{
			"audience", "market", "segment", "demographic",
			"consumer", "customer", "buyer", "user",
			"group", "target", "population", "adults",
			"kids", "teens", "families", "professionals",
		},

		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productAudience(f), nil
		},
	})

	AddFuncLookup("productdimension", Info{
		Display:     "Product Dimension",
		Category:    "product",
		Description: "The size or dimension of a product",
		Example:     "medium",
		Output:      "string",
		Aliases: []string{
			"product size",
			"product measurement",
			"item dimensions",
			"product scale",
			"size specification",
		},
		Keywords: []string{
			"dimension", "size", "measurement", "proportion",
			"scale", "specification", "specs", "length",
			"width", "height", "depth", "volume", "weight",
			"product", "item",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productDimension(f), nil
		},
	})

	AddFuncLookup("productusecase", Info{
		Display:     "Product Use Case",
		Category:    "product",
		Description: "The scenario or purpose for which a product is typically used",
		Example:     "home",
		Output:      "string",
		Aliases: []string{
			"use case",
			"product purpose",
			"intended use",
			"product application",
			"usage scenario",
		},
		Keywords: []string{
			"use", "usecase", "purpose", "usage", "application",
			"context", "scenario", "situation", "case",
			"intention", "goal", "objective", "function",
			"home", "office", "outdoor", "industrial", "commercial",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productUseCase(f), nil
		},
	})

	AddFuncLookup("productbenefit", Info{
		Display:     "Product Benefit",
		Category:    "product",
		Description: "The key advantage or value the product provides",
		Example:     "comfort",
		Output:      "string",
		Aliases: []string{
			"product advantage",
			"product value",
			"user benefit",
			"customer gain",
			"selling point",
		},
		Keywords: []string{
			"benefit", "advantage", "value", "improvement",
			"enhancement", "feature", "positive", "outcome",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productBenefit(f), nil
		},
	})

	AddFuncLookup("productsuffix", Info{
		Display:     "Product Suffix",
		Category:    "product",
		Description: "A suffix used to differentiate product models or versions",
		Example:     "pro",
		Output:      "string",
		Aliases: []string{
			"product suffix",
			"model suffix",
			"version suffix",
			"edition suffix",
			"name suffix",
		},
		Keywords: []string{
			"suffix", "variant", "edition", "version", "model",
			"series", "line", "tier", "release", "upgrade",
			"plus", "pro", "max", "lite", "mini",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productSuffix(f), nil
		},
	})

	AddFuncLookup("productisbn", Info{
		Display:     "Product ISBN",
		Category:    "product",
		Description: "ISBN-10 or ISBN-13 identifier for books",
		Example:     "978-1-4028-9462-6",
		Output:      "string",
		Aliases: []string{
			"isbn code", "isbn number", "book isbn", "isbn13",
			"isbn10", "publication code", "book identifier",
		},
		Keywords: []string{
			"identifier", "publication", "library", "catalog",
			"literature", "reference", "edition", "registration", "publishing",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return productISBN(f, nil), nil
		},
	})
}
