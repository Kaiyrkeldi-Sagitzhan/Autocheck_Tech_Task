// Command gen-dataset produces a deterministic 1C-style CSV export with
// 1000+ automobile records, including intentionally problematic rows.
package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"awesomeProject5/internal/parser"
)

// seed is fixed for deterministic output.
const seed = 42

// totalRecords is the number of valid records to generate (excluding problematic ones).
const totalRecords = 1000

func main() {
	parser.Seed(seed)
	r := rand.New(rand.NewSource(seed))

	brands := []string{
		"Toyota", "Hyundai", "Kia", "Volkswagen", "Renault",
		"Lada", "BMW", "Mercedes-Benz", "Audi", "Lexus",
		"Mazda", "Nissan", "Honda", "Chevrolet", "Ford",
		"Skoda", "Peugeot", "Mitsubishi", "Subaru", "Volvo",
	}

	models := map[string][]string{
		"Toyota":        {"Camry", "Corolla", "RAV4", "Land Cruiser", "Highlander", "Hilux"},
		"Hyundai":       {"Tucson", "Creta", "Santa Fe", "Elantra", "Sonata", "Palisade"},
		"Kia":           {"Sportage", "Rio", "Seltos", "Sorento", "K5", "Carnival"},
		"Volkswagen":    {"Polo", "Tiguan", "Passat", "Touareg", "Golf", "Jetta"},
		"Renault":       {"Duster", "Kaptur", "Arkana", "Megane", "Logan", "Sandero"},
		"Lada":          {"Granta", "Vesta", "XRAY", "Niva", "Largus", "Priora"},
		"BMW":           {"X5", "X3", "5 Series", "3 Series", "X1", "7 Series"},
		"Mercedes-Benz": {"E-Class", "C-Class", "GLE", "GLC", "S-Class", "A-Class"},
		"Audi":          {"Q5", "A4", "Q7", "A6", "Q3", "A8"},
		"Lexus":         {"RX", "LX", "ES", "NX", "GX", "LS"},
		"Mazda":         {"CX-5", "CX-9", "Mazda6", "CX-30", "CX-4", "MX-5"},
		"Nissan":        {"Qashqai", "X-Trail", "Patrol", "Navara", "Juke", "Murano"},
		"Honda":         {"CR-V", "Civic", "Accord", "HR-V", "Pilot", "Fit"},
		"Chevrolet":     {"Malibu", "Equinox", "Tahoe", "Traverse", "Camaro", "Spark"},
		"Ford":          {"Focus", "Explorer", "Mustang", "Escape", "F-150", "Edge"},
		"Skoda":         {"Octavia", "Kodiaq", "Karoq", "Superb", "Fabia", "Scala"},
		"Peugeot":       {"3008", "2008", "5008", "508", "308", "Partner"},
		"Mitsubishi":    {"Outlander", "Pajero", "ASX", "Lancer", "Eclipse Cross", "Pajero Sport"},
		"Subaru":        {"Forester", "Outback", "XV", "Legacy", "WRX", "BRZ"},
		"Volvo":         {"XC60", "XC90", "S60", "V60", "XC40", "S90"},
	}

	colors := []string{
		"Белый", "Черный", "Серебристый", "Красный", "Синий",
		"Серый", "Зеленый", "Коричневый", "Оранжевый", "Желтый",
		"Бежевый", "Фиолетовый", "Голубой", "Бордовый", "Хаки",
	}

	transmissions := []string{"AT", "MT", "CVT", "DCT", "AMT"}
	bodyTypes := []string{"Седан", "Хэтчбек", "SUV", "Кроссовер", "Универсал", "Пикап", "Минивэн", "Купе", "Кабриолет", "Лифтбэк"}
	engines := []string{
		"2.0 бензин 150 л.с.", "1.6 бензин 110 л.с.", "2.5 бензин 186 л.с.",
		"3.5 бензин 249 л.с.", "1.8 гибрид 122 л.с.", "2.0 дизель 150 л.с.",
		"1.4 турбо 140 л.с.", "2.0 турбо 240 л.с.", "3.0 дизель 245 л.с.",
		"1.6 дизель 116 л.с.", "2.4 бензин 173 л.с.", "5.0 бензин 422 л.с.",
	}
	defectsList := []string{
		"", "", "", "", "", "", "", "", "", "", // mostly no defects
		"Царапина на бампере", "Проблемы с АКПП", "Двигатель троит",
		"Требуется замена масла", "Стук в подвеске", "Не работает кондиционер",
		"Повреждение лакокрасочного покрытия", "Износ тормозных колодок",
		"Утечка антифриза", "Шум в коробке передач", "Проблемы с электроникой",
		"Требуется замена ремня ГРМ", "Подвески требуют обслуживания",
	}

	statuses := []string{"active", "sold", "reserved"}

	// Generate VINs deterministically.
	vinChars := "ABCDEFGHJKLMNPRSTUVWXYZ0123456789"

	records := make([][]string, 0, totalRecords+10)
	records = append(records, []string{
		"VIN", "Brand", "Model", "Year", "MileageKm", "Price",
		"Currency", "Color", "Engine", "Transmission", "BodyType",
		"DefectsRaw", "Status", "UpdatedAt",
	})

	problematicRows := map[int]string{
		5:   "missing_vin",
		50:  "invalid_mileage",
		100: "invalid_year",
		200: "missing_brand",
		500: "malformed_columns",
		750: "missing_model",
	}

	for i := 0; i < totalRecords; i++ {
		rowNum := i + 2 // 1-based with header

		if problemType, ok := problematicRows[rowNum]; ok {
			records = append(records, generateProblematicRow(problemType, r, vinChars))
			continue
		}

		brand := brands[r.Intn(len(brands))]
		modelList := models[brand]
		model := modelList[r.Intn(len(modelList))]

		year := 2000 + r.Intn(26) // 2000-2025
		mileage := r.Intn(200000)
		price := 500000 + r.Intn(15000000) // 500k - 15.5M KZT

		color := colors[r.Intn(len(colors))]
		transmission := transmissions[r.Intn(len(transmissions))]
		bodyType := bodyTypes[r.Intn(len(bodyTypes))]
		engine := engines[r.Intn(len(engines))]
		defects := defectsList[r.Intn(len(defectsList))]
		status := statuses[r.Intn(len(statuses))]

		// Generate a valid VIN (17 chars, no I/O/Q)
		vin := generateVIN(r, vinChars)

		// UpdatedAt: random date in last 2 years
		updatedAt := randomDate(r)

		records = append(records, []string{
			vin, brand, model, strconv.Itoa(year), strconv.Itoa(mileage),
			strconv.Itoa(price), "KZT", color, engine, transmission,
			bodyType, defects, status, updatedAt,
		})
	}

	// Write CSV
	f, err := os.Create("data/imports/partner_1c_export.csv")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = ';'
	w.UseCRLF = true

	if err := w.WriteAll(records); err != nil {
		fmt.Fprintf(os.Stderr, "write csv: %v\n", err)
		os.Exit(1)
	}
	w.Flush()

	if err := w.Error(); err != nil {
		fmt.Fprintf(os.Stderr, "flush csv: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %d rows (%d data + %d header) to data/imports/partner_1c_export.csv\n",
		len(records)-1, len(records)-1, 1)
}

func generateVIN(r *rand.Rand, chars string) string {
	vin := make([]byte, 17)
	for i := range vin {
		vin[i] = chars[r.Intn(len(chars))]
	}
	return string(vin)
}

func randomDate(r *rand.Rand) string {
	// Random date within last 730 days
	daysAgo := r.Intn(730)
	t := time.Now().AddDate(0, 0, -daysAgo)
	return t.Format("2006-01-02")
}

func generateProblematicRow(problemType string, r *rand.Rand, vinChars string) []string {
	base := []string{
		"TOYOTA", "Camry", "2020", "50000", "5000000",
		"KZT", "Белый", "2.0 бензин 150 л.с.", "AT", "Седан",
		"", "active", "2024-01-01",
	}

	switch problemType {
	case "missing_vin":
		base[0] = "" // missing VIN
	case "invalid_mileage":
		base[3] = "abc" // invalid mileage
	case "invalid_year":
		base[2] = "1800" // invalid year
	case "missing_brand":
		base[1] = "" // missing brand
	case "missing_model":
		base[2] = "" // missing model (shifted)
		base[1] = "Corolla"
	case "malformed_columns":
		// Too few columns
		return []string{"SHORT", "Row"}
	}

	return base
}
